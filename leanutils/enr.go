package leanutils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

type ENR struct {
	Record *enr.Record
}

func NewENR() (*ENR, error) {
	return &ENR{
		Record: &enr.Record{},
	}, nil
}

func (e *ENR) Decode(raw string) error {
	b := []byte(raw)
	if bytes.HasPrefix(b, []byte("enr:")) {
		b = b[4:]
	}

	dec := make([]byte, base64.RawURLEncoding.DecodedLen(len(b)))

	n, err := base64.RawURLEncoding.Decode(dec, b)
	if err != nil {
		return err
	}

	var r enr.Record
	err = rlp.DecodeBytes(dec[:n], &r)
	if err != nil {
		return err
	}

	e.Record = &r
	return nil
}

func (e *ENR) Encode() (string, error) {
	if e.Record == nil {
		return "", fmt.Errorf("record is nil")
	}

	buf := new(bytes.Buffer)

	err := e.Record.EncodeRLP(buf)
	if err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	encoded = "enr:" + encoded

	return encoded, nil
}

// Set sets arbitrary key-value pairs in the ENR record
func (e *ENR) Set(entries ...enr.Entry) {
	for _, entry := range entries {
		e.Record.Set(entry)
	}
}

// SetIP sets the IP address in the ENR record
func (e *ENR) SetIP(ip net.IP) {
	if ip4 := ip.To4(); ip4 != nil {
		e.Record.Set(enr.IPv4(ip4))
	} else if ip6 := ip.To16(); ip6 != nil {
		e.Record.Set(enr.IPv6(ip6))
	}
}

// SetIP4 explicitly sets an IPv4 address
func (e *ENR) SetIP4(ip net.IP) {
	if ip4 := ip.To4(); ip4 != nil {
		e.Record.Set(enr.IPv4(ip4))
	}
}

// SetIP6 explicitly sets an IPv6 address
func (e *ENR) SetIP6(ip net.IP) {
	if ip6 := ip.To16(); ip6 != nil {
		e.Record.Set(enr.IPv6(ip6))
	}
}

// SetTCP sets the TCP port in the ENR record
func (e *ENR) SetTCP(port int) {
	e.Record.Set(enr.TCP(port))
}

// SetUDP sets the UDP port in the ENR record
func (e *ENR) SetUDP(port int) {
	e.Record.Set(enr.UDP(port))
}

// SetTCP6 sets the IPv6-specific TCP port
func (e *ENR) SetTCP6(port int) {
	e.Record.Set(enr.TCP6(port))
}

// SetUDP6 sets the IPv6-specific UDP port
func (e *ENR) SetUDP6(port int) {
	e.Record.Set(enr.UDP6(port))
}

// SetEntry sets a custom key-value pair in the ENR record
func (e *ENR) SetEntry(key string, value interface{}) {
	e.Record.Set(enr.WithEntry(key, value))
}

// Sign signs the ENR record with the provided private key
func (e *ENR) Sign(privKey *ecdsa.PrivateKey) error {
	return enode.SignV4(e.Record, privKey)
}

// SetSeq sets a specific sequence number
func (e *ENR) SetSeq(seq uint64) {
	e.Record.SetSeq(seq)
}

// Load retrieves a value from the ENR record by key
func (e *ENR) Load(entry enr.Entry) error {
	return e.Record.Load(entry)
}

// LoadEntry loads a custom entry from the ENR record
func (e *ENR) LoadEntry(key string, value interface{}) error {
	return e.Record.Load(enr.WithEntry(key, value))
}

// Seq returns the sequence number of the ENR record
func (e *ENR) Seq() uint64 {
	return e.Record.Seq()
}

// NodeAddr returns the node address (IP and UDP port) from the ENR
func (e *ENR) NodeAddr() (net.IP, int, error) {
	var ip net.IP
	var udpPort enr.UDP

	// Try IPv4 first
	var ip4 enr.IPv4
	if err := e.Record.Load(&ip4); err == nil {
		ip = net.IP(ip4)
	} else {
		// Try IPv6
		var ip6 enr.IPv6
		if err := e.Record.Load(&ip6); err == nil {
			ip = net.IP(ip6)
		} else {
			return nil, 0, fmt.Errorf("no IP address in ENR")
		}
	}

	if err := e.Record.Load(&udpPort); err != nil {
		return nil, 0, fmt.Errorf("no UDP port in ENR")
	}

	return ip, int(udpPort), nil
}

// PublicKey extracts the public key from a signed ENR record
func (e *ENR) PublicKey() (*ecdsa.PublicKey, error) {
	var pubkey enode.Secp256k1
	if err := e.Record.Load(&pubkey); err != nil {
		return nil, err
	}
	return (*ecdsa.PublicKey)(&pubkey), nil
}

// Verify verifies the ENR signature
func (e *ENR) Verify() error {
	_, err := enode.New(enode.ValidSchemes, e.Record)
	return err
}
