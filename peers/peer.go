package peers

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/AcidOP/torrly/messages"
)

type Peer struct {
	IP       net.IP
	Port     int
	choked   bool
	conn     net.Conn
	Bitfield []bool
}

// Read function reads a `messages.Message` from the peer's connection.
// (Optionally) accepts a timeout duration to set a read deadline.
// If no timeout is provided, it defaults to 2 minutes.
func (p *Peer) Read(timeout ...time.Duration) (*messages.Message, error) {
	duration := 120 * time.Second
	if len(timeout) > 0 {
		duration = timeout[0]
	}

	p.conn.SetReadDeadline(time.Now().Add(duration))
	defer p.conn.SetReadDeadline(time.Time{})

	return messages.Receive(p.conn)
}

// ReadLoop continuously reads messages from the peer until an error occurs.
// This call blocks until a message is received or an error occurs.
func (p *Peer) ReadLoop() error {
	for {
		msg, err := p.Read(time.Second * 10)
		if err != nil {
			p.conn.Close()
			return err
		}

		switch msg.ID {
		case messages.MsgKeepAlive:
			fmt.Println("Received keep-alive message from peer:", p.IP.String())
			continue
		case messages.MsgBitfield:
			p.setBitfield(bytesToBoolSlice(msg.Payload))
		case messages.MsgChoke:
			// p.choke()
			fmt.Printf("Peer %s has choked us\n", p.IP.String())
		case messages.MsgUnchoke:
			p.unchoke()
			fmt.Printf("Peer %s has unchoked us\n", p.IP.String())
		case messages.MsgHave:
			fmt.Printf("Peer %s has piece %d\n", p.IP.String(), len(msg.Payload))
		case messages.MsgPiece:
			fmt.Printf("Received %d bytes from peer %s\n", len(msg.Payload), p.IP.String())
		case messages.MsgNotInterested:
			fmt.Printf("Peer %s is not interested\n", p.IP.String())
		default:
			return fmt.Errorf("unknown message ID %d from peer %s", msg.ID, p.IP.String())
		}
	}
}

func (p *Peer) download(pieceIndex, pieceLength int) error {
	if p.choked {
		fmt.Println("Peer has choked us, skipping.")
		return fmt.Errorf("peer has choked us")
	}

	fmt.Printf("Requesting piece %d from peer %s\n", pieceIndex, p.IP.String())

	blockSize := 16 * 1024 // 16 KB
	for begin := 0; begin < pieceLength; begin += blockSize {
		length := blockSize
		if begin+length > pieceLength {
			length = pieceLength - begin
		}

		if err := p.SendRequest(pieceIndex, length, begin); err != nil {
			return err
		}

		msg, err := p.Read()
		if err != nil {
			return err
		}

		if msg.ID != messages.MsgPiece {
			fmt.Printf("\n[%s] Expected piece message, but got ID %d\n", p.IP.String(), msg.ID)
			continue
		}

		fmt.Printf("\n[%s] Received %d bytes for piece %d\n", p.IP.String(), len(msg.Payload), pieceIndex)
	}
	return nil
}

func (p *Peer) choke() {
	p.choked = true
	fmt.Printf("[Peer %s] Choked\n", p.IP.String())
}

func (p *Peer) unchoke() {
	p.choked = false
	fmt.Printf("[Peer %s] Unchoked\n", p.IP.String())
}

func (p *Peer) setBitfield(bf []bool) error {
	p.Bitfield = append(p.Bitfield, bf...)
	return nil
}

// bytesToBoolSlice helper func converts a []byte bitfield to a []bool slice.
func bytesToBoolSlice(bf []byte) []bool {
	bools := make([]bool, 0, len(bf)*8)
	for _, b := range bf {
		for i := 7; i >= 0; i-- {
			bools = append(bools, (b>>i)&1 == 1)
		}
	}
	return bools
}

// COnnect to the associated peer using its IP and Port.
// Attaches the connection to the `peer` struct which MUST
// be closed by the caller later in the program.
func (p *Peer) Connect() error {
	ipStr := p.IP.String()
	portStr := strconv.Itoa(p.Port)
	addr := net.JoinHostPort(ipStr, portStr)

	// pick family based on parsed IP
	var tryOrder []string
	if p.IP == nil {
		// fallback to generic tcp (let resolver decide) if IP unknown
		tryOrder = []string{"tcp"}
	} else if p.IP.To4() != nil {
		// IPv4 address
		tryOrder = []string{"tcp4", "tcp"}
	} else {
		// IPv6 address: try IPv6 first, then optionally try IPv4 fallback (only if sensible)
		tryOrder = []string{"tcp6", "tcp4", "tcp"}
	}

	var conn net.Conn
	var lastErr error
	for _, network := range tryOrder {
		c, err := net.DialTimeout(network, addr, 5*time.Second)
		if err == nil {
			conn = c
			break
		}
		lastErr = err
		// If this was an IPv6 attempt and the error indicates "network unreachable",
		// continue to try the next network.
	}

	if conn == nil {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("failed to dial peer %s", addr)
	}

	p.conn = conn
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Connected to peer: %s\n", p.IP.String())
	fmt.Println(strings.Repeat("-", 50))

	return nil
}
