package rtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrame(t *testing.T) {
	frame := &Packet{
		Seq:       1234,
		Timestamp: 5678,
		Extension: &Extension{
			Profile:          123,
			Length:           2,
			HeaderExtensions: make([]byte, 8),
		},
	}

	bytes := frame.Encode()
	newFrame := &Packet{}
	code := newFrame.Decode(bytes)

	assert.True(t, code == len(bytes))
	assert.True(t, newFrame.X == frame.X)
	assert.True(t, newFrame.Seq == frame.Seq)
	assert.True(t, newFrame.Timestamp == frame.Timestamp)
	assert.True(t, newFrame.Extension.Profile == frame.Extension.Profile)
	assert.True(t, newFrame.Extension.Length == frame.Extension.Length)
}

func TestBasic(t *testing.T) {
	p := &Packet{}

	if code := p.Decode([]byte{}); code > 0 {
		t.Fatal("Unmarshal did not error on zero length packet")
	}

	rawPkt := []byte{
		0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
		0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
	}

	code := p.Decode(rawPkt)
	assert.True(t, code > 0)

	data := p.Encode()
	assert.True(t, len(rawPkt) == len(data))
	for i, b := range data {
		assert.True(t, b == rawPkt[i])
	}

	parsedPacket := &Packet{
		V:      2,
		P:      0,
		Marker: 1,
		X:      1,
		Extension: &Extension{
			Profile:          1,
			Length:           1,
			HeaderExtensions: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},

		PT:        96,
		Seq:       27023,
		Timestamp: 3653407706,
		SSRC:      476325762,
		CSRC:      []uint32{},
		Payload:   rawPkt[20:],
	}

	bytes := parsedPacket.Encode()
	assert.True(t, len(rawPkt) == len(bytes))
	for i, b := range rawPkt {
		assert.True(t, b == bytes[i])
	}
}
