package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/yingshulu/rtp"
)

type Conn struct {
	addr    *net.UDPAddr
	readCh  chan []byte
	conn    *net.UDPConn
	timeout time.Duration
	buffer  bytes.Buffer
	closed  bool
}

func NewClientConn(addr *net.UDPAddr, conn *net.UDPConn) *Conn {
	return &Conn{
		addr:    addr,
		conn:    conn,
		timeout: 10 * time.Second,
		readCh:  make(chan []byte, 20),
		buffer:  *bytes.NewBuffer(nil),
	}
}

func (c *Conn) Read(buf []byte) (int, error) {
	if c.closed {
		return 0, errors.New("udp client closed")
	}

	if c.buffer.Len() == 0 {
		select {
		case <-time.After(c.timeout):
			return 0, errors.New("timeout")

		case data := <-c.readCh:
			_, err := c.buffer.Write(data)
			if err != nil {
				return 0, err
			}
		}
	}

	fmt.Println("server Conn read buf: ", c.buffer.Len())
	fmt.Println("server Conn read buffer size: ", len(buf))
	n, err := c.buffer.Read(buf)
	if err != nil {
		fmt.Println("server Conn buf truncate: ", err)
	}

	fmt.Println("server Conn buf left: ", c.buffer.Len())

	return n, err
}

func (c *Conn) Write(data []byte) (int, error) {
	if c.closed {
		return 0, errors.New("udp client closed")
	}
	return c.conn.WriteToUDP(data, c.addr)
}

func (c *Conn) Close() error {
	c.closed = true
	return nil
}

func processFrame(f *rtp.Frame) {

}

func handleRtp(c *Conn) {
	pc := rtp.NewConn(c, c.timeout)
	s := pc.Stream(1234)

	exec := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		f, err := s.ReadFrame(ctx)
		if f != nil {
			processFrame(f)
		} else {
			//skip this frame?
			return
		}

		if err != nil {
			fmt.Println("read frame error ", err)
			return
		}
	}

	for {
		exec()
	}
}

type UdpServer struct {
	closed  bool
	clients map[string]*Conn
}

func NewUdpServer() *UdpServer {
	return &UdpServer{
		clients: map[string]*Conn{},
	}
}

func (u *UdpServer) Serve(addr string) error {
	listenAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return err
	}

	for !u.closed {
		buf := make([]byte, 1500)
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if n > 0 {
			fmt.Println("udp server read buffer: ", n)
		}
		if err != nil {
			continue
		}

		fmt.Println("udp recv buffer: ", n)

		cc, ok := u.clients[clientAddr.String()]
		if !ok {
			cc = NewClientConn(clientAddr, conn)
			u.clients[clientAddr.String()] = cc
			go handleRtp(cc)
		}
		cc.readCh <- buf[:n]
	}
	return nil
}

func (u *UdpServer) Close() error {
	u.closed = true
	for _, c := range u.clients {
		c.Close()
	}
	return nil
}

func main() {
	u := NewUdpServer()
	err := u.Serve("127.0.0.1:8765")
	if err != nil {
		fmt.Println("udp server failure: ", err)
	}
}
