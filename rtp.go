package rtp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type Conn interface {
	Stream(uint32) Stream
	Close() error
}

func NewConn(io io.ReadWriteCloser, timeout time.Duration) Conn {
	c := &conn{
		ReadWriteCloser: io,
		timeout:         timeout,
		streams:         map[uint32]Stream{},
		writeCh:         make(chan *Packet, 100),
		dispatchCh:      make(chan dispatchItem, 100),
	}

	var ctx context.Context
	ctx, c.done = context.WithCancel(context.Background())

	go c.readPump()
	go c.writePump(ctx)
	go c.dispatchPump(ctx)
	return c
}

type dispatchItem struct {
	s Stream
	p *Packet
}

type conn struct {
	sync.Mutex
	io.ReadWriteCloser
	timeout time.Duration
	streams map[uint32]Stream

	writeCh    chan *Packet
	dispatchCh chan dispatchItem
	done       context.CancelFunc
	closed     bool
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
	var buff = make([]byte, 1500)
	for !c.closed {
		n, err := c.Read(buff)
		if err != nil {
			fmt.Print("read error: ", err)
			break
		}

		p := &Packet{}
		code := p.Decode(buff[:n])
		if code < 0 {
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

		fmt.Println("packet seq: ", p.Seq)
		c.dispatchCh <- dispatchItem{
			s: s,
			p: p,
		}
	}
}

func (c *conn) dispatchPump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case d := <-c.dispatchCh:
			err := d.s.dispatch(d.p)
			if err != nil {
				fmt.Println("stream dispatch error: ", err)
			}
		}
	}
}

func (c *conn) writePacket(p *Packet) error {
	if c.closed {
		return errors.New("udp conn closed")
	}
	c.writeCh <- p
	return nil
}

func (c *conn) writePump(ctx context.Context) {
	defer c.Close()

	for {
		select {
		case <-ctx.Done():
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
	c.done()
	return c.ReadWriteCloser.Close()
}
