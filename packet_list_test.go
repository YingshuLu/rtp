package rtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsert(t *testing.T) {
	exec := func(seqs []uint16, isFull bool) {
		var packets []*Packet
		for _, seq := range seqs {
			packets = append(packets, &Packet{
				Seq: seq,
			})
		}

		pl := NewPacketList()
		for _, p := range packets {
			pl.Insert(p)
		}
		assert.False(t, pl.IsFull())

		var results []uint16
		for c := pl.NewCursor(); ; {
			p := c.Next()
			if p == nil {
				break
			}
			results = append(results, p.Seq)
		}
		t.Log(results)

		pl.Last().Marker = 1
		if isFull {
			assert.True(t, pl.IsFull())
		}
	}

	testCases := []struct {
		seqs   []uint16
		isFull bool
	}{
		{
			seqs:   []uint16{0, 1, 65533, 5, 4, 2, 3, 65534, 6, 65535, 10, 3, 8, 6, 65531, 7, 9, 8, 9, 65532, 65530, 1, 7, 9},
			isFull: true,
		},
		{
			seqs:   []uint16{10, 1, 15, 90, 46, 2, 10, 45, 90, 490},
			isFull: false,
		},
	}

	for _, item := range testCases {
		exec(item.seqs, item.isFull)
	}
}
