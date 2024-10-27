package rtp

import (
	"encoding/binary"
	"fmt"
)

const FixedHeaderSize = 12

const (
	Ok      int = iota
	Lack        = -1
	Illegal     = -2
)

/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |      defined by profile       |           length              |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                        header extension                       |
   |                             ....                              |
*/

type Extension struct {
	Profile          uint16
	Length           uint16
	HeaderExtensions []byte
}

func (e *Extension) Count() int {
	return int(4*e.Length + 4)
}

func (e *Extension) Encode() []byte {
	size := 4*int(e.Length) + 4
	data := make([]byte, 0, size)

	data = binary.BigEndian.AppendUint16(data, e.Profile)
	data = binary.BigEndian.AppendUint16(data, e.Length)
	data = append(data, e.HeaderExtensions...)
	return data
}

func (e *Extension) Decode(data []byte) int {
	n := len(data)
	if n < 4 {
		return Lack
	}

	e.Profile = binary.BigEndian.Uint16(data)
	e.Length = binary.BigEndian.Uint16(data[2:])

	expected := int(e.Length)*4 + 4
	if n < expected {
		return Lack
	}

	e.HeaderExtensions = data[4:expected]
	return expected
}

/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |V=2|P|X|  CC   |M|     PT      |       sequence number         |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                           timestamp                           |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |           synchronization source (SSRC) identifier            |
   +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
   |            contributing source (CSRC) identifiers             |
   |                             ....                              |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type Packet struct {
	V         byte
	P         byte
	X         byte
	CC        byte
	Marker    byte
	PT        byte
	Seq       uint16
	Timestamp uint32
	SSRC      uint32
	CSRC      []uint32
	Extension *Extension
	Payload   []byte
}

func (f *Packet) Encode() []byte {
	f.V = byte(2)
	if f.Extension != nil {
		f.X = byte(1)
	}
	f.CC = byte(len(f.CSRC))

	size := FixedHeaderSize + 4*len(f.CSRC) + len(f.Payload)
	if f.Extension != nil {
		size += f.Extension.Count()
	}

	data := make([]byte, 0, size)

	firstByte := f.V<<6 | f.P<<5 | f.X<<4 | f.CC
	data = append(data, firstByte)

	secondByte := f.Marker<<7 | f.PT
	data = append(data, secondByte)

	data = binary.BigEndian.AppendUint16(data, f.Seq)
	data = binary.BigEndian.AppendUint32(data, f.Timestamp)
	data = binary.BigEndian.AppendUint32(data, f.SSRC)

	for _, c := range f.CSRC {
		data = binary.BigEndian.AppendUint32(data, c)
	}

	if f.Extension != nil {
		data = append(data, f.Extension.Encode()...)
	}

	data = append(data, f.Payload...)
	return data
}

func (f *Packet) Decode(data []byte) int {
	if len(data) < FixedHeaderSize {
		return Lack
	}

	firstByte := data[0]
	f.V = firstByte >> 6
	f.P = (firstByte & 0x20) >> 5
	f.X = (firstByte & 0x10) >> 4
	f.CC = firstByte & 0x0f

	secondByte := data[1]
	f.Marker = (secondByte & 0x80) >> 7
	f.PT = secondByte & 0x7f

	f.Seq = binary.BigEndian.Uint16(data[2:])
	f.Timestamp = binary.BigEndian.Uint32(data[4:])
	f.SSRC = binary.BigEndian.Uint32(data[8:])

	cc := int(f.CC)
	if len(data[FixedHeaderSize:]) < cc*4 {
		return Lack
	}

	f.CSRC = make([]uint32, cc)
	index := FixedHeaderSize
	for i := 0; i < cc; i++ {
		f.CSRC[i] = binary.BigEndian.Uint32(data[index:])
		index += 4
	}

	extensionSize := 0
	if f.X == 1 {
		f.Extension = &Extension{}
		status := f.Extension.Decode(data[index:])
		if status < 0 {
			return status
		}
		extensionSize = status
	}

	start := index + extensionSize
	end := len(data)

	// padding
	if f.P == 1 {
		end -= int(data[end-1])
	}
	if end < start {
		return Illegal
	}

	f.Payload = data[start:end]
	return len(data)
}

func (f *Packet) String() string {
	return fmt.Sprintf("V: %d\nX: %d\nCC: %d\nMarker: %d\nPT: %d\nSeq: %d\nTimestamp: %d\nSSRC: %d\n",
		f.V, f.X, f.CC, f.Marker, f.PT, f.Seq, f.Timestamp, f.SSRC)
}
