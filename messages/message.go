package messages

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MsgID = uint8

const (
	MsgKeepAlive     MsgID = 255
	MsgChoke         MsgID = 0
	MsgUnchoke       MsgID = 1
	MsgInterested    MsgID = 2
	MsgNotInterested MsgID = 3
	MsgHave          MsgID = 4
	MsgBitfield      MsgID = 5
	MsgRequest       MsgID = 6
	MsgPiece         MsgID = 7
	MsgCancel        MsgID = 8
)

type Message struct {
	ID      MsgID
	Payload []byte
}

// Serialize converts the `Message` into a byte slice for peer communication.
// Syntax: <length prefix><message ID><payload>.
// https://wiki.theory.org/BitTorrentSpecification#Messages
func (msg *Message) Serialize() []byte {
	length := len(msg.Payload) + 1 // +1 for the message ID
	buf := make([]byte, 4+length)

	binary.BigEndian.PutUint32(buf[0:4], uint32(length))

	buf[4] = msg.ID

	copy(buf[5:], msg.Payload)
	return buf
}

func (msg *Message) String() string {
	switch msg.ID {
	case MsgKeepAlive:
		return "Keep Alive"
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgNotInterested:
		return "Not Interested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgRequest:
		return "Request"
	case MsgPiece:
		return "Piece"
	case MsgCancel:
		return "Cancel"
	default:
		return fmt.Sprintf("Unknown Message ID: %d", msg.ID)
	}
}

// Receive function reads data from a stream and returns a `Message`.
// It expects the first 4 bytes to be the length of the message,
// followed by the message ID and payload.
func Receive(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// FIX: Handle Keep-Alive
	if length == 0 {
		return &Message{ID: MsgKeepAlive, Payload: nil}, nil
	} else {
		id := make([]byte, 1)
		if _, err := io.ReadFull(r, id); err != nil {
			return nil, err
		}
		msgID := MsgID(id[0])

		payloadLength := int(length) - 1
		payload := make([]byte, payloadLength)

		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}

		msg := &Message{
			ID:      msgID,
			Payload: payload,
		}

		return msg, nil
	}
}
