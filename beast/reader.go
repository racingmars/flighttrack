// Package beast decodes a stream of binary messages from the format described at
// https://wiki.jetvision.de/wiki/Mode-S_Beast:Data_Output_Formats
package beast

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
)

// Type describes the transponder message format.
type Type int

const (
	// ModeAC is a 2-byte Mode A/C message
	ModeAC Type = 1

	// ModeSshort is a 7-byte Mode S message
	ModeSshort Type = 2

	// ModeSlong is a 14-byte Mode S message
	ModeSlong Type = 3
)

// Reader converts the binary messages in an io.Reader to Message structs.
type Reader struct {
	bufrdr *bufio.Reader
}

// Message is a single transponder message decoded from the beast format.
type Message struct {
	Type        Type
	Timestamp   []byte
	SignalLevel byte
	Message     []byte
}

// UnknownFormatError indicates that the beast message type was not a known type.
type UnknownFormatError error

// New creates a new beast decoder on an io.Reader.
func New(rdr io.Reader) *Reader {
	return &Reader{bufrdr: bufio.NewReader(rdr)}
}

// Read will return the next message from the beast stream.
func (r *Reader) Read() (*Message, error) {
	// Advance until escape character
	var charBuf byte
	var err error
	for charBuf != 0x1a {
		charBuf, err = r.getByte()
		if err != nil {
			return nil, err
		}
	}

	// Type
	charBuf, err = r.getByte()
	if err != nil {
		return nil, err
	}

	msg := new(Message)

	var data []byte
	switch charBuf {
	case '1':
		msg.Type = ModeAC
		data, err = r.getBytes(9)
	case '2':
		msg.Type = ModeSshort
		data, err = r.getBytes(14)
	case '3':
		msg.Type = ModeSlong
		data, err = r.getBytes(21)
	default:
		return nil, UnknownFormatError(fmt.Errorf("unexpected message type %s",
			hex.EncodeToString([]byte{charBuf})))
	}
	if err != nil {
		return nil, err
	}

	msg.Timestamp = data[0:6]
	msg.SignalLevel = data[6]
	msg.Message = data[7:]

	return msg, nil
}

// getByte returns the next byte from the Reader, un-escaping the 0x1a
// escape sequence if necessary.
func (r *Reader) getByte() (byte, error) {
	b, err := r.bufrdr.ReadByte()
	if err != nil {
		return b, err
	}
	if b == 0x1a {
		p, err := r.bufrdr.Peek(1)
		if err == io.EOF {
			return b, nil
		}
		if err != nil {
			return b, err
		}
		if p[0] == 0x1a {
			r.bufrdr.Discard(1)
		}
	}
	return b, nil
}

// getBytes returns the next `numBytes` un-escaped bytes from the Reader.
func (r *Reader) getBytes(numBytes int) ([]byte, error) {
	bytes := make([]byte, numBytes)
	pos := 0
	for pos < len(bytes) {
		n, err := r.bufrdr.Read(bytes[pos:])
		if err != nil {
			return nil, err
		}
		pos += n
	}
	return bytes, nil
}
