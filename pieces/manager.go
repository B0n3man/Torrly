package pieces

import "fmt"

type bitfield []bool

type PieceManager struct {
	Pieces      []*Piece
	Bitfield    bitfield // Our very own bitfield to track completed pieces
	Length      int
	PieceLength int
	Pending     map[int]bool // Tracks pieces that are currently being downloaded
}

func NewPieceManager(hashes []hash, bfield []bool, length, pieceLength int) *PieceManager {
	pieces := make([]*Piece, len(hashes))

	for i, v := range hashes {
		p := Piece{index: i, hash: v}
		pieces[i] = &p
	}

	pm := &PieceManager{
		Pieces:      pieces,
		Length:      length,
		PieceLength: pieceLength,
	}

	return pm
}

func (pm *PieceManager) NextPiece(peerBField bitfield) *Piece {
	for i := 0; i < len(pm.Pieces); i++ {
		if peerBField.Has(i) && !pm.Bitfield.Has(i) && !pm.Pending[i] {
			return pm.Pieces[i]
		}
	}
	return nil
}

func (pm *PieceManager) MarkComplete(piece *Piece) error {
	if piece.index < 0 || piece.index >= len(pm.Pieces) {
		return fmt.Errorf("invalid piece index: %d", piece.index)
	}

	if !piece.Verified {
		return fmt.Errorf("cannot mark piece %d complete: not verified", piece.index)
	}

	pm.Bitfield[piece.index] = true

	return nil
}

func (bf bitfield) Has(index int) bool {
	if index < 0 || index > len(bf) {
		return false
	}
	return bf[index]
}
