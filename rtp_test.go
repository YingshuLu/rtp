package rtp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {

	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8765")
	assert.Nil(t, err)

	conn, err := net.DialUDP("udp", nil, serverAddr)
	assert.Nil(t, err)

	data := []byte{
		0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
		0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
	}

	c := NewConn(conn, 100*time.Microsecond)
	s := c.Stream(1234)
	n, err := s.WriteFrame(data, 0x2, 3000, nil)
	assert.Nil(t, err)

	t.Log("write frame data: ", n)
}
