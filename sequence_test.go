package rtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Sequencer_Next(t *testing.T) {
	s := NewRandomSequencer()
	res := map[uint16]bool{}
	count := 100
	for i := 0; i < count; i++ {
		res[s.Next()] = true
	}

	assert.True(t, len(res) == count)
}
