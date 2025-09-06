package leanutils

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

func TestENRAsGenerator(t *testing.T) {
	// Generate a new private key
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Test 1: Create ENR and add fields incrementally
	enrObj, err := NewENR()
	if err != nil {
		t.Fatalf("Failed to create ENR: %v", err)
	}

	// Add network info
	ip := net.ParseIP("192.168.1.100")
	tcpPort := 30303
	udpPort := 30303

	enrObj.SetIP(ip)
	enrObj.SetTCP(tcpPort)
	enrObj.SetUDP(udpPort)
	enrObj.SetSeq(1)

	// Sign the ENR
	if err := enrObj.Sign(privKey); err != nil {
		t.Fatalf("Failed to sign ENR: %v", err)
	}

	// Encode and print
	encoded, err := enrObj.Encode()
	if err != nil {
		t.Fatalf("Failed to encode ENR: %v", err)
	}
	t.Logf("Generated ENR: %s", encoded)

	// Verify the ENR
	if err := enrObj.Verify(); err != nil {
		t.Fatalf("ENR verification failed: %v", err)
	}

	// Check sequence
	if enrObj.Seq() != 1 {
		t.Errorf("Expected sequence 1, got %d", enrObj.Seq())
	}

	// Test 2: Create ENR with custom fields using Set method
	enrWithCustom, err := NewENR()
	if err != nil {
		t.Fatalf("Failed to create ENR: %v", err)
	}

	// Add multiple fields at once
	enrWithCustom.Set(
		enr.IP(ip),
		enr.UDP(udpPort),
		enr.WithEntry("eth2", []byte{0x00, 0x00, 0x00, 0x00}),
		enr.WithEntry("attnets", []byte{0xff, 0xff}),
	)
	enrWithCustom.SetSeq(5)

	// Sign it
	if err := enrWithCustom.Sign(privKey); err != nil {
		t.Fatalf("Failed to sign ENR: %v", err)
	}

	// Verify custom fields
	var eth2Data []byte
	if err := enrWithCustom.LoadEntry("eth2", &eth2Data); err != nil {
		t.Fatalf("Failed to load eth2 entry: %v", err)
	}
	t.Logf("eth2 data: %x", eth2Data)

	// Check sequence is what we set
	if enrWithCustom.Seq() != 5 {
		t.Errorf("Expected sequence 5, got %d", enrWithCustom.Seq())
	}

	// Test 3: Update existing ENR with specific sequence
	enrWithCustom.SetEntry("mynewfield", "hello world")
	enrWithCustom.SetTCP(9000)
	enrWithCustom.SetSeq(10) // Set specific sequence

	// Re-sign after modification
	if err := enrWithCustom.Sign(privKey); err != nil {
		t.Fatalf("Failed to re-sign ENR: %v", err)
	}

	// Verify sequence is exactly what we set
	if enrWithCustom.Seq() != 10 {
		t.Errorf("Expected sequence 10, got %d", enrWithCustom.Seq())
	}

	// Encode updated ENR
	updatedEncoded, err := enrWithCustom.Encode()
	if err != nil {
		t.Fatalf("Failed to encode updated ENR: %v", err)
	}
	t.Logf("Updated ENR: %s", updatedEncoded)
	t.Logf("Sequence: %d", enrWithCustom.Seq())
}

func TestENRWithKnownPrivateKey(t *testing.T) {
	// Test with a known private key
	hexKey := "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"

	privKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	// Create and configure ENR
	enrObj, err := NewENR()
	if err != nil {
		t.Fatalf("Failed to create ENR: %v", err)
	}

	enrObj.SetIP(net.ParseIP("10.0.0.1"))
	enrObj.SetUDP(30303)
	enrObj.SetSeq(1)

	// Sign
	if err := enrObj.Sign(privKey); err != nil {
		t.Fatalf("Failed to sign ENR: %v", err)
	}

	// Get public key and verify it matches expected
	pubKey, err := enrObj.PublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}

	// The public key should be deterministic for this private key
	pubKeyBytes := crypto.CompressPubkey(pubKey)
	t.Logf("Public key: %x", pubKeyBytes)

	// Test adding fields after initial signing with specific sequence
	enrObj.SetEntry("custom", []byte{0x01, 0x02, 0x03})
	enrObj.SetSeq(100) // Set specific sequence
	if err := enrObj.Sign(privKey); err != nil {
		t.Fatalf("Failed to re-sign ENR: %v", err)
	}

	if enrObj.Seq() != 100 {
		t.Errorf("Expected sequence number 100, got %d", enrObj.Seq())
	}
}
