package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/AcidOP/torrly/peers"
	"github.com/jackpal/bencode-go"
)

type hash = [20]byte

type Torrent struct {
	Name        string
	Announce    string
	InfoHash    hash
	PieceHashes []hash // Array of 20-byte hashes for each piece
	PieceLength int    // Number of bytes in each piece (e.g. 16 KB)
	Length      int    // Total length of the file in bytes
	PeerId      string // Our own Peer ID, used for handshakes.
	Port        int    // Port we listen on for incoming connections
}

type bcodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`       // Concatenated SHA1 hashes of each piece
	PieceLength int    `bencode:"piece length"` // Length of each piece in bytes (e.g. 16 KB)
	Length      int    `bencode:"length"`       // Total length of the file in bytes
}

type bcodeTorrent struct {
	Announce string    `bencode:"announce"`
	Info     bcodeInfo `bencode:"info"`
}

const (
	PeerID = "-TRLY01-123456789012"
	Port   = 6881
)

func NewTorrentFromFile(path string) (*Torrent, error) {
	file, err := parseTorrentFromPath(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return metaFromFile(file)
}

// Visualize information about the torrent.
// (e.g. announce URL, file name, size, piece length, number of pieces, info hash)
func (t *Torrent) ViewTorrent() {
	var displaySize string

	// Format the size in GB, MB, or KB
	if s := t.Length / (1024 * 1024); s >= 1024 {
		displaySize = fmt.Sprintf("%.2f GB", float64(s)/1024)
	} else if s >= 1 {
		displaySize = fmt.Sprintf("%.2f MB", float64(s))
	} else {
		displaySize = fmt.Sprintf("%d KB", t.Length/1024)
	}

	line := strings.Repeat("#", 70)

	fmt.Println()
	fmt.Println(line)
	fmt.Printf("\nAnnounce: %s\n", t.Announce)
	fmt.Printf("File name: %s\n", t.Name)
	fmt.Printf("File size: %s\n", displaySize)
	fmt.Printf("Piece length: %d KB\n", t.PieceLength/1024)
	fmt.Printf("Number of pieces: %d\n", len(t.PieceHashes))
	fmt.Printf("Info Hash: %x\n\n", t.InfoHash)
	fmt.Println(line)
	fmt.Println()
}

func (t *Torrent) FirstTwentyHashes() {
	fmt.Println("First 20 piece hashes:")
	for i := 0; i < 20 && i < len(t.PieceHashes); i++ {
		fmt.Printf("%d: %x\n", i, t.PieceHashes[i])
	}
}

func (t *Torrent) StartDownload() {
	pArr, err := t.GetAvailablePeers()
	if err != nil {
		fmt.Println(err)
		return
	}

	pm := peers.NewPeerManager(
		pArr,
		t.InfoHash[:],
		[]byte(t.PeerId),
	)
	pm.HandlePeers()
}

// Takes a path as an argument and checks if the file is a .torrent file.
// Then reads the file and a pointer to the file.
func parseTorrentFromPath(path string) (*os.File, error) {
	f, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, err
	}

	// Check the extension
	if strings.Split(f.Name(), ".")[1] != "torrent" {
		return nil, errors.New("file " + f.Name() + " is not a .torrent file")
	}

	return os.Open(path)
}

// Takes a file as argument and reads the torrent metadata from it.
// Returns a Torrent struct with the metadata.
func metaFromFile(f *os.File) (*Torrent, error) {
	if f == nil {
		return nil, errors.New("file pointer is nil")
	}

	bt := bcodeTorrent{}
	if err := bencode.Unmarshal(f, &bt); err != nil {
		return nil, errors.New("failed to parse torrent file: " + err.Error())
	}

	// SHA1 hash of `info` dictionary
	iHash := bt.Info.hash()

	// Split the pieces into an array of  hashes
	pHashes, err := bt.Info.splitPieceHashes()
	if err != nil {
		return nil, err
	}

	t := &Torrent{
		Announce:    bt.Announce,
		InfoHash:    iHash,
		PieceHashes: pHashes,
		PieceLength: bt.Info.PieceLength,
		Length:      bt.Info.Length,
		Name:        bt.Info.Name,
		PeerId:      PeerID,
		Port:        Port,
	}
	return t, nil
}

// Calculate the SHA1 hash of the bencoded info dictionary.
func (i bcodeInfo) hash() hash {
	infoBytes := bytes.Buffer{}
	if err := bencode.Marshal(&infoBytes, i); err != nil {
		panic("failed to marshal info: " + err.Error())
	}

	return sha1.Sum(infoBytes.Bytes())
}

// Take the `info` key from meta and split the pieces into an array of hashes.
// Returns an array of 20-byte hashes.
func (i bcodeInfo) splitPieceHashes() ([]hash, error) {
	hashLen := 20 // SHA1 is 20 bytes long

	if len(i.Pieces)%hashLen != 0 {
		return nil, errors.New("malformed pieces: " + fmt.Sprint(len(i.Pieces)))
	}

	numHashes := len(i.Pieces) / hashLen
	expectedNumHashes := int(math.Ceil(float64(i.Length) / float64(i.PieceLength)))

	if numHashes != expectedNumHashes {
		return nil, fmt.Errorf("piece count mismatch: got %d hashes, expected %d (file size=%d, piece size=%d)",
			numHashes, expectedNumHashes, i.Length, i.PieceLength)
	}

	hashes := make([]hash, numHashes)

	for idx := 0; idx < numHashes; idx++ {
		copy(hashes[idx][:], i.Pieces[idx*hashLen:(idx+1)*hashLen])
	}
	return hashes, nil
}
