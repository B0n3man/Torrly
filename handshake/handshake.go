package handshake

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	HANDSHAKE_LENGTH = 68
	PROTOCOL_LENGTH  = 19
	PROTOCOL_STRING  = "BitTorrent protocol"
	RESERVED_LENGTH  = 8
	HASH_LENGTH      = 20
	PEER_ID_LENGTH   = 20
)

// https://wiki.theory.org/BitTorrentSpecification#Handshake
type Handshake struct {
	InfoHash  []byte
	PeerID    []byte
	pLength   int
	pStr      string
	pReserved []byte
}

func NewHandshake(infoHash, peerID []byte) (*Handshake, error) {
	if len(infoHash) != HASH_LENGTH {
		return nil, fmt.Errorf("info hash must be %d bytes, got %d", HASH_LENGTH, len(infoHash))
	}

	return &Handshake{
		pLength:   PROTOCOL_LENGTH,
		pStr:      PROTOCOL_STRING,
		pReserved: make([]byte, RESERVED_LENGTH), // All zeros
		InfoHash:  infoHash,
		PeerID:    peerID,
	}, nil
}

func (h *Handshake) ExchangeHandshake(connPeer net.Conn) ([]byte, error) {
	hBuf := bytes.Buffer{}

	// Build handshake: pstrlen + pstr + reserved + info_hash + peer_id
	hBuf.WriteByte(byte(h.pLength)) // pstrlen (1 byte)
	hBuf.WriteString(h.pStr)        // pstr (19 bytes)
	hBuf.Write(h.pReserved)         // reserved (8 bytes) - should be all zeros
	hBuf.Write(h.InfoHash)          // info_hash (20 bytes)
	hBuf.Write(h.PeerID)            // peer_id (20 bytes)

	hBytes := hBuf.Bytes()

	if len(hBytes) != HANDSHAKE_LENGTH {
		return nil, fmt.Errorf("handshake byte length expected %d bytes, got: %d",
			HANDSHAKE_LENGTH, len(hBytes))
	}

	// Proper debugging output
	fmt.Printf("\n\n=== OUTGOING HANDSHAKE ===\n")
	fmt.Printf("Total length: %d bytes\n", len(hBytes))
	fmt.Printf("Protocol length: %d\n", hBytes[0])
	fmt.Printf("Protocol string: %q\n", string(hBytes[1:20]))
	fmt.Printf("Reserved field: %x (should be all zeros)\n", hBytes[20:28])
	fmt.Printf("Info hash: %x\n", hBytes[28:48])
	fmt.Printf("Peer ID: %q\n", string(hBytes[48:68]))
	fmt.Printf("Full handshake: %s\n", hBytes)
	fmt.Printf("========================\n")

	if _, err := connPeer.Write(hBytes); err != nil {
		return nil, fmt.Errorf("failed to send handshake: %v", err)
	}

	// Set read timeout
	connPeer.SetReadDeadline(time.Now().Add(time.Second * 10))

	received := make([]byte, HANDSHAKE_LENGTH)
	if _, err := io.ReadFull(connPeer, received); err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}

	// Proper debugging for received handshake
	fmt.Printf("\n\n=== INCOMING HANDSHAKE ===\n")
	fmt.Printf("Total length: %d bytes\n", len(received))
	fmt.Printf("Protocol length: %d\n", received[0])
	fmt.Printf("Protocol string: %q\n", string(received[1:20]))
	fmt.Printf("Reserved field: %x\n", received[20:28])
	fmt.Printf("Info hash: %x\n", received[28:48])
	fmt.Printf("Peer ID: %q\n", string(received[48:68]))
	fmt.Printf("Full handshake: %s\n", received)
	fmt.Printf("========================\n")

	return received, nil
}

// Takes a Connection (to another peer) as an argument and sends our handshake.
// Then waits for the peer to respond with its handshake and return it
func (h *Handshake) ExchangeHandshakeOld(connPeer net.Conn) ([]byte, error) {
	hBuf := bytes.Buffer{}

	// Build handshake: pstrlen + pstr + reserved + info_hash + peer_id
	hBuf.WriteByte(byte(h.pLength)) // pstrlen (1 byte)
	hBuf.WriteString(h.pStr)        // pstr (19 bytes)
	hBuf.Write(h.pReserved)         // reserved (8 bytes)
	hBuf.Write(h.InfoHash)          // info_hash (20 bytes)
	hBuf.Write(h.PeerID)            // peer_id (20 bytes)

	hBytes := hBuf.Bytes() // 1 + 19 + 8 + 20 + 20 = 68 bytes

	if len(hBytes) != HANDSHAKE_LENGTH {
		return nil, fmt.Errorf("handshake byte length expected %d bytes, got: %d",
			HANDSHAKE_LENGTH, len(hBytes))
	}

	fmt.Printf("Sending Handshake (%d bytes): %s\n", len(hBytes), hBytes)

	if _, err := connPeer.Write(hBytes); err != nil {
		return nil, fmt.Errorf("failed to send handshake: %v", err)
	}

	// Set read timeout
	connPeer.SetReadDeadline(time.Now().Add(time.Second * 10))

	received := make([]byte, HANDSHAKE_LENGTH)
	if _, err := io.ReadFull(connPeer, received); err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}

	fmt.Printf("Received Handshake (%d bytes): %s\n", len(received), received)

	return received, nil
}

// Decode a Handshake sent by another Peer
func DecodeHandshake(buf []byte) (*Handshake, error) {
	if len(buf) != HANDSHAKE_LENGTH {
		return nil, fmt.Errorf("invalid handshake length, expected %d bytes, got: %d",
			HANDSHAKE_LENGTH, len(buf))
	}

	pLength := int(buf[0])
	if pLength != PROTOCOL_LENGTH {
		return nil, fmt.Errorf("invalid protocol length: expected %d, got %d",
			PROTOCOL_LENGTH, pLength)
	}

	pStr := string(buf[1 : 1+pLength])
	if pStr != PROTOCOL_STRING {
		return nil, fmt.Errorf("invalid protocol string: expected %q, got %q",
			PROTOCOL_STRING, pStr)
	}

	h := &Handshake{
		pLength:   pLength,
		pStr:      pStr,
		pReserved: buf[20:28], // 8 bytes
		InfoHash:  buf[28:48], // 20 bytes
		PeerID:    buf[48:68], // 20 bytes
	}

	return h, nil
}

func (h *Handshake) VerifyHandshake(raw []byte) error {
	h2, err := DecodeHandshake(raw)
	if err != nil {
		return fmt.Errorf("failed to decode handshake: %v", err)
	}

	if h2.pLength != PROTOCOL_LENGTH {
		return fmt.Errorf("protocol length mismatch: expected %d, got %d",
			PROTOCOL_LENGTH, h2.pLength)
	}

	if h2.pStr != PROTOCOL_STRING {
		return fmt.Errorf("protocol string mismatch: expected %q, got %q",
			PROTOCOL_STRING, h2.pStr)
	}

	if len(h2.pReserved) != RESERVED_LENGTH {
		return fmt.Errorf("reserved field length mismatch: expected %d, got %d",
			RESERVED_LENGTH, len(h2.pReserved))
	}

	if !bytes.Equal(h.InfoHash, h2.InfoHash) {
		return fmt.Errorf("info hash mismatch: expected %x, got %x",
			h.InfoHash, h2.InfoHash)
	}

	return nil
}
