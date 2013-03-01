package streetsphere

import (
	"bufio"
	"encoding/binary"
	"io"
)

type Marker byte

const (
	APP1 Marker = Marker(0xE1)
)

type Section struct {
	Data   []byte
	Offset int
}

// NextSection finds the next section denoted by marker m.
func NextSection(r *bufio.Reader, m Marker) (*Section, error) {
	var prev byte

	offset := 0

	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		if prev == 0xFF && b == byte(m) {
			return readSection(r, offset)
		}

		prev = b
		offset++
	}
	return nil, nil
}

// readSection reads the contents of the section at the current position of r.
func readSection(r io.Reader, offset int) (*Section, error) {
	s := Section{
		Offset: offset + 2,
	}

	var size uint16
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}

	size -= 2 // marker is two bytes long.

	s.Data = make([]byte, size)

	if _, err := io.ReadFull(r, s.Data); err != nil {
		return nil, err
	}

	return &s, nil
}
