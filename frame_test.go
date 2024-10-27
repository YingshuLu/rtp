package rtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FrameQueue(t *testing.T) {
	exec := func(timestamps []uint32) {
		q := NewFrameQueue()

		unique := map[uint32]bool{}
		for _, tm := range timestamps {
			f := NewFrame(nil)
			f.Push(&Packet{
				Timestamp: tm,
			})
			q.Push(f)
			unique[tm] = true
		}

		var (
			prev  *Frame
			count int
		)
		for !q.Empty() {
			f := q.Pop()
			count += 1
			assert.True(t, f.prevFrame == prev)
			prev = f
		}
		assert.True(t, count == len(unique))
	}

	testCases := [][]uint32{
		{5000, 9000, 4000, 7000, 10000, 3000, 2000, 6000, 4000},
		{4567, 8906, 2341, 4801, 3421, 2340, 9012},
	}

	for _, item := range testCases {
		exec(item)
	}
}
