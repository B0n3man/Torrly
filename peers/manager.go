package peers

import (
	"fmt"
	"sync"

	"github.com/AcidOP/torrly/handshake"
	"github.com/AcidOP/torrly/messages"
)

type PeerManager struct {
	peers          []Peer
	infoHash       []byte
	peerId         []byte
	connectedPeers []*Peer
}

func NewPeerManager(peers []Peer, infoHash, peerId []byte) *PeerManager {
	return &PeerManager{
		peers:    peers,
		infoHash: infoHash,
		peerId:   peerId,
	}
}

func (pm *PeerManager) HandlePeers() {
	hs, err := handshake.NewHandshake(pm.infoHash, pm.peerId)
	if err != nil {
		fmt.Println("Error creating handshake:", err)
		return
	}

	var wg sync.WaitGroup

	// This does the error thing
	handleErr := func(p *Peer, err error, msg string) bool {
		if err != nil {
			fmt.Printf("%s: %v\n", msg, err)
			if p.conn != nil {
				p.conn.Close()
			}
			return true
		}
		return false
	}

	for i := range pm.peers {
		peer := &pm.peers[i]

		if handleErr(peer, peer.Connect(), "Error connecting to peer") {
			continue
		}

		if handleErr(peer, hs.ExchangeHandshake(peer.conn), fmt.Sprintf("Handshake failed with: %s", peer.IP.String())) {
			continue
		}

		if handleErr(peer, pm.AddPeer(peer), fmt.Sprintf("Error adding peer %s", peer.IP.String())) {
			continue
		}

		wg.Add(1)
		go func(p *Peer) {
			defer wg.Done()
			p.SendInterested()
			if err := p.ReadLoop(); err != nil {
				handleErr(p, err, fmt.Sprintf("Error reading from peer %s", p.IP.String()))
			}
		}(peer)
	}

	wg.Wait()
}

func (pm *PeerManager) AddPeer(p *Peer) error {
	if p.IP == nil || p.Port <= 0 || p.Port > 65535 || p.conn == nil {
		return fmt.Errorf("invalid peer: %v", p)
	}

	// Check if the peer already exists
	for _, existingPeer := range pm.connectedPeers {
		if existingPeer.IP.Equal(p.IP) {
			return fmt.Errorf("peer already exists: %s", p.IP)
		}
	}

	pm.connectedPeers = append(pm.connectedPeers, p)
	return nil
}

func (pm *PeerManager) RemovePeer(p Peer) error {
	for i, existingPeer := range pm.peers {
		if existingPeer.IP.Equal(p.IP) {
			if existingPeer.conn != nil {
				existingPeer.conn.Close()
			}
			pm.peers = append(pm.peers[:i], pm.peers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("peer not found: %s", p.IP)
}

func (pm *PeerManager) BroadcastMessage(msg *messages.Message) {
	for _, peer := range pm.connectedPeers {
		if err := peer.send(msg); err != nil {
			fmt.Printf("Error sending message to peer %s: %v\n", peer.IP.String(), err)
			continue
		}

		fmt.Printf("Broadcasted message (%s) to peer: %s\n", msg.String(), peer.IP.String())
	}
}
