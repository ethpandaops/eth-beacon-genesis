package validators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const xmssPubkeySize = 32

type XMSSPubkeySource struct {
	CacheDir string
	URL      string
}

func mnemonicFingerprint(seed []byte) string {
	hash := sha256.Sum256(append([]byte(xmssSeedDomain), seed...))
	return hex.EncodeToString(hash[:8])
}

func (s *XMSSPubkeySource) fileName(seed []byte) string {
	return mnemonicFingerprint(seed) + ".bin"
}

func (s *XMSSPubkeySource) pubkeys(seed []byte, start, count uint64) ([][32]byte, error) {
	pubkeys := make([][32]byte, count)

	if s.CacheDir != "" {
		if err := s.readCache(seed, start, pubkeys); err != nil {
			return nil, fmt.Errorf("failed to read xmss pubkey cache: %w", err)
		}
	}

	missing := countMissing(pubkeys)
	cached := int(count) - missing //nolint:gosec // validator counts fit in int

	if missing > 0 && s.URL != "" {
		if err := s.fetch(seed, start, pubkeys); err != nil {
			logrus.Warnf("failed to fetch xmss pubkeys, deriving them instead: %v", err)
		}
	}

	fetched := missing - countMissing(pubkeys)
	derived := 0

	for i := range pubkeys {
		if pubkeys[i] != [32]byte{} {
			continue
		}

		pubkey, err := deriveXMSSPubkey(seed, start+uint64(i))
		if err != nil {
			return nil, err
		}

		pubkeys[i] = pubkey
		derived++

		if derived%10 == 0 {
			logrus.Infof("...derived leanXMSS key %d/%d", derived, missing-fetched)
		}
	}

	logrus.Infof("leanXMSS keys: %d from cache, %d fetched, %d derived", cached, fetched, derived)

	if s.CacheDir != "" && fetched+derived > 0 {
		if err := s.writeCache(seed, start, pubkeys); err != nil {
			return nil, fmt.Errorf("failed to write xmss pubkey cache: %w", err)
		}
	}

	return pubkeys, nil
}

func countMissing(pubkeys [][32]byte) int {
	missing := 0

	for i := range pubkeys {
		if pubkeys[i] == [32]byte{} {
			missing++
		}
	}

	return missing
}

func fillMissing(pubkeys [][32]byte, data []byte) {
	for i := range pubkeys {
		if pubkeys[i] == [32]byte{} && len(data) >= (i+1)*xmssPubkeySize {
			copy(pubkeys[i][:], data[i*xmssPubkeySize:])
		}
	}
}

func (s *XMSSPubkeySource) readCache(seed []byte, start uint64, pubkeys [][32]byte) error {
	file, err := os.Open(filepath.Join(s.CacheDir, s.fileName(seed)))
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	defer file.Close()

	data := make([]byte, len(pubkeys)*xmssPubkeySize)

	n, err := file.ReadAt(data, int64(start)*xmssPubkeySize) //nolint:gosec // key indexes fit in int64
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	fillMissing(pubkeys, data[:n])

	return nil
}

func (s *XMSSPubkeySource) writeCache(seed []byte, start uint64, pubkeys [][32]byte) error {
	if err := os.MkdirAll(s.CacheDir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(s.CacheDir, s.fileName(seed)), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer file.Close()

	data := make([]byte, 0, len(pubkeys)*xmssPubkeySize)
	for i := range pubkeys {
		data = append(data, pubkeys[i][:]...)
	}

	_, err = file.WriteAt(data, int64(start)*xmssPubkeySize) //nolint:gosec // key indexes fit in int64

	return err
}

func (s *XMSSPubkeySource) fetch(seed []byte, start uint64, pubkeys [][32]byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(s.URL, "/")+"/"+s.fileName(seed), http.NoBody)
	if err != nil {
		return err
	}

	first := start * xmssPubkeySize
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", first, first+uint64(len(pubkeys))*xmssPubkeySize-1))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusPartialContent:
	case http.StatusOK:
		if uint64(len(data)) < first {
			return nil
		}

		data = data[first:]
	case http.StatusRequestedRangeNotSatisfiable:
		return nil
	default:
		return fmt.Errorf("unexpected status %s for %s", resp.Status, req.URL)
	}

	fillMissing(pubkeys, data)

	return nil
}
