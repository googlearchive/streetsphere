package streetsphere

import (
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

// readByte reads the next byte from r.
func readByte(r io.Reader) (byte, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	return b[0], err
}

// NextSection finds the next section denoted by marker m.
func NextSection(r io.Reader, m Marker) (s *Section, err error) {
	var prev byte
	var b byte

	offset := 0

	for {
		b, err = readByte(r)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return
		}

		if prev == 0xFF && b == byte(m) {
			return makeSection(r, offset)
		}

		prev = b
		offset++
	}
	return
}

// makeSection reads the contents of the section at the current position of r.
func makeSection(r io.Reader, offset int) (s *Section, err error) {
	s = new(Section)
	s.Offset = offset + 2

	var size uint16
	if err = binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}

	size -= 2 // marker is two bytes long.

	s.Data = make([]byte, int(size))

	if _, err = io.ReadFull(r, s.Data); err != nil {
		return nil, err
	}

	return
}
