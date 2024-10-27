package rtp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const MTU = 1300

/*
	RTP Session
		- Stream
			- Frame
				- Packet
*/

type Stream interface {
	dispatch(p *Packet) error
	ReadFrame(ctx context.Context) (*Frame, error)
	WriteFrame(payload []byte, typ byte, samples uint32, csrc []uint32) (int, error)
	SkipSamples(uint32)
	SSRC() uint32
}

func NewStream(ssrc uint32, timeout time.Duration, sendPacket func(*Packet) error) Stream {
	return &stream{
		frameQueue: NewFrameQueue(),
		frameMap:   make(map[uint32]*Frame),
		timeout:    timeout,
		sendPacket: sendPacket,
		ssrc:       ssrc,
		sequencer:  NewRandomSequencer(),
	}
}

type stream struct {
	frameQueue FrameQueue

	mutex sync.Mutex

	frameMap map[uint32]*Frame

	timeout time.Duration

	currFrame *Frame

	sendPacket func(*Packet) error

	sequencer Sequencer

	timestamp uint32

	ssrc uint32
}

func (s *stream) SSRC() uint32 {
	return s.ssrc
}

func (s *stream) dispatch(p *Packet) error {
	if p.SSRC != s.ssrc {
		return errors.New("packet not SSRC stream")
	}
	timeout := p.Timestamp

	var f *Frame
	curr := s.currFrame
	if curr != nil {
		if timeout < curr.Timestamp() {
			return errors.New("packet too old")
		} else if timeout == curr.Timestamp() {
			f = curr
		}
	}

	if f == nil {
		s.mutex.Lock()

		f = s.frameMap[timeout]
		if f == nil {
			f = NewFrame(nil)
		}
		s.frameMap[timeout] = f
		s.frameQueue.Push(f)

		// heading
		if f.prevFrame == nil && curr != nil {
			f.prevFrame = curr
		}

		s.mutex.Unlock()
	}

	if ok := f.Push(p); ok != AcceptOk {
		return fmt.Errorf("frame push failed: %v", ok)
	}
	return nil
}

func (s *stream) ReadFrame(ctx context.Context) (*Frame, error) {
	s.mutex.Lock()
	f := s.frameQueue.Pop()
	s.mutex.Unlock()

	if f == nil {
		return nil, errors.New("no packets")
	}

	defer func() {
		s.mutex.Lock()
		delete(s.frameMap, f.Timestamp())
		s.mutex.Unlock()
	}()

	s.currFrame = f
	select {
	case <-ctx.Done():
		return f, ctx.Err()
	case <-time.After(s.timeout):
		return f, errors.New("read frame timeout")
	case <-f.Done():
		return f, nil
	}
}

func (s *stream) WriteFrame(payload []byte, typ byte, samples uint32, csrc []uint32) (int, error) {
	var (
		start, end int
		sent       int
		err        error
	)

	defer func() {
		s.timestamp += samples
	}()

	for left := len(payload); left > 0; left = len(payload) {
		p := &Packet{
			PT:        typ,
			Seq:       s.sequencer.Next(),
			Timestamp: s.timestamp,
			SSRC:      s.ssrc,
			CSRC:      csrc,
		}

		if left > MTU {
			end = start + MTU
		} else {
			end = start + left
			p.Marker = 1
		}

		p.Payload = payload[start:end]
		start = end
		payload = payload[end:]

		err = s.sendPacket(p)
		if err != nil {
			break
		}
		sent += len(p.Payload)
	}
	return sent, err
}

func (s *stream) SkipSamples(samples uint32) {
	s.timestamp += samples
}
