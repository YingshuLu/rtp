package rtp

import (
	"math/rand"
	"sync"
)

type Sequencer interface {
	Next() uint16
	Round() uint32
}

func NewRandomSequencer() Sequencer {
	pos := rand.Uint32() % 31
	leftByte := uint16((rand.Uint32() >> pos) & 0xff)
	rightByte := uint16((rand.Uint32() >> (32 - pos)) & 0xff)

	id := (leftByte << 4) | rightByte
	return &sequence{
		counter: id,
	}
}

type sequence struct {
	rwmutex sync.RWMutex
	counter uint16
	round   uint32
}

func (s *sequence) Next() uint16 {
	s.rwmutex.Lock()
	defer s.rwmutex.Unlock()

	s.counter += 1
	if s.counter == 0 {
		s.round += 1
	}
	return s.counter
}

func (s *sequence) Round() uint32 {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()

	return s.round
}
