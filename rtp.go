package rtp

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type Conn interface {
	Stream(uint32) Stream
}

func NewConn(io io.ReadWriteCloser, timeout time.Duration) Conn {
	c := &conn{
		ReadWriteCloser: io,
		timeout:         timeout,
		streams:         map[uint32]Stream{},
		writeCh:         make(chan *Packet, 100),
		done:            make(chan bool, 1),
	}
	go c.readPump()
	go c.writePump()
	return c
}

type conn struct {
	sync.Mutex
	io.ReadWriteCloser
	timeout time.Duration
	streams map[uint32]Stream

	writeCh chan *Packet
	done    chan bool
	closed  bool
}

func (c *conn) Stream(ssrc uint32) Stream {
	c.Lock()
	defer c.Unlock()
	s := c.streams[ssrc]
	if s == nil {
		s = NewStream(ssrc, c.timeout, c.writePacket)
		c.streams[ssrc] = s
	}
	return s
}

func (c *conn) readPump() {
	for {
		var buff = make([]byte, 1500)
		n, err := c.Read(buff)
		if err != nil {
			fmt.Print("read error: ", err)
			break
		}

		p := &Packet{}
		code := p.Decode(buff[:n])
		if code != Ok {
			fmt.Print("packet parse error: ", code)
			break
		}

		ssrc := p.SSRC
		c.Lock()
		s := c.streams[ssrc]
		if s == nil {
			s = NewStream(ssrc, c.timeout, c.writePacket)
			c.streams[ssrc] = s
		}
		c.Unlock()

		go func(s Stream, p *Packet) {
			err = s.dispatch(p)
			if err != nil {
				fmt.Print("stream dispatch error: ", err)
			}
		}(s, p)
	}
}

func (c *conn) writePacket(p *Packet) error {
	if c.closed {
		return errors.New("udp conn closed")
	}
	c.writeCh <- p
	return nil
}

func (c *conn) writePump() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return

		case p := <-c.writeCh:
			data := p.Encode()
			_, err := c.Write(data)
			if err != nil {

				return
			}
		}
	}

}

func (c *conn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.done <- true
	return c.ReadWriteCloser.Close()
}
