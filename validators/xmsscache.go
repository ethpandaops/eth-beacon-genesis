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
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	xmssRecordSize    = 64
	xmssKeyFileDomain = "leanxmss-keccak-vrf-keys"
)

type XMSSPubkeySource struct {
	CacheDir string
	URL      string
}

func mnemonicFingerprint(seed []byte) string {
	hash := sha256.Sum256(append([]byte(xmssKeyFileDomain), seed...))
	return hex.EncodeToString(hash[:8])
}

func (s *XMSSPubkeySource) fileName(seed []byte) string {
	return mnemonicFingerprint(seed) + ".bin"
}

func (s *XMSSPubkeySource) keys(seed []byte, start, count uint64) ([]PQKeyMaterial, error) {
	keys := make([]PQKeyMaterial, count)

	if s.CacheDir != "" {
		if err := s.readCache(seed, start, keys); err != nil {
			return nil, fmt.Errorf("failed to read xmss key cache: %w", err)
		}
	}

	missing := countMissing(keys)
	cached := int(count) - missing //nolint:gosec // validator counts fit in int

	if missing > 0 && s.URL != "" {
		if err := s.fetch(seed, start, keys); err != nil {
			logrus.Warnf("failed to fetch xmss keys, deriving them instead: %v", err)
		}
	}

	fetched := missing - countMissing(keys)
	derived := missing - fetched

	if derived > 0 {
		if err := deriveMissing(seed, start, keys); err != nil {
			return nil, err
		}
	}

	logrus.Infof("leanXMSS keys: %d from cache, %d fetched, %d derived", cached, fetched, derived)

	if s.CacheDir != "" && fetched+derived > 0 {
		if err := s.writeCache(seed, start, keys); err != nil {
			return nil, fmt.Errorf("failed to write xmss key cache: %w", err)
		}
	}

	return keys, nil
}

func deriveMissing(seed []byte, start uint64, keys []PQKeyMaterial) error {
	var g errgroup.Group

	g.SetLimit(runtime.NumCPU())

	total := countMissing(keys)
	derived := 0

	for i := range keys {
		if keys[i].XMSSPublicKey != [32]byte{} {
			continue
		}

		idx := start + uint64(i)

		g.Go(func() error {
			root, err := deriveVRFRoot(seed, idx)
			if err != nil {
				return err
			}

			keys[i].VRFRoot = root

			return nil
		})

		pubkey, err := deriveXMSSPubkey(seed, idx)
		if err != nil {
			return err
		}

		keys[i].XMSSPublicKey = pubkey
		derived++

		if derived%10 == 0 {
			logrus.Infof("...derived leanXMSS key %d/%d", derived, total)
		}
	}

	return g.Wait()
}

func countMissing(keys []PQKeyMaterial) int {
	missing := 0

	for i := range keys {
		if keys[i].XMSSPublicKey == [32]byte{} {
			missing++
		}
	}

	return missing
}

func fillMissing(keys []PQKeyMaterial, data []byte) {
	for i := range keys {
		if keys[i].XMSSPublicKey == [32]byte{} && len(data) >= (i+1)*xmssRecordSize {
			copy(keys[i].XMSSPublicKey[:], data[i*xmssRecordSize:])
			copy(keys[i].VRFRoot[:], data[i*xmssRecordSize+32:])
		}
	}
}

func (s *XMSSPubkeySource) readCache(seed []byte, start uint64, keys []PQKeyMaterial) error {
	file, err := os.Open(filepath.Join(s.CacheDir, s.fileName(seed)))
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	defer file.Close()

	data := make([]byte, len(keys)*xmssRecordSize)

	n, err := file.ReadAt(data, int64(start)*xmssRecordSize) //nolint:gosec // key indexes fit in int64
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	fillMissing(keys, data[:n])

	return nil
}

func (s *XMSSPubkeySource) writeCache(seed []byte, start uint64, keys []PQKeyMaterial) error {
	if err := os.MkdirAll(s.CacheDir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(s.CacheDir, s.fileName(seed)), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer file.Close()

	data := make([]byte, 0, len(keys)*xmssRecordSize)
	for i := range keys {
		data = append(data, keys[i].XMSSPublicKey[:]...)
		data = append(data, keys[i].VRFRoot[:]...)
	}

	_, err = file.WriteAt(data, int64(start)*xmssRecordSize) //nolint:gosec // key indexes fit in int64

	return err
}

func (s *XMSSPubkeySource) fetch(seed []byte, start uint64, keys []PQKeyMaterial) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(s.URL, "/")+"/"+s.fileName(seed), http.NoBody)
	if err != nil {
		return err
	}

	first := start * xmssRecordSize
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", first, first+uint64(len(keys))*xmssRecordSize-1))

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

	fillMissing(keys, data)

	return nil
}
