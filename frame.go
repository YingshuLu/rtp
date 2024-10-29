package rtp

import (
	"context"
	"sync"
)

const (
	prevFrameNone = iota
	prevFrameDrain
	prevFrameOk
)

func NewFrame(prevFrame *Frame) *Frame {
	return &Frame{
		PacketList: NewPacketList(),
		done:       make(chan bool),
		prevFrame:  prevFrame,
	}
}

type Frame struct {
	PacketList
	prevFrame *Frame
	done      chan bool
}

func (f *Frame) Done() <-chan bool {
	return f.done
}

func (f *Frame) Timestamp() uint32 {
	if f.First() == nil {
		return 0
	}
	return f.First().Timestamp
}

// Push Packet
func (f *Frame) Push(p *Packet) int {
	if p == nil {
		return DenyPacketNil
	}

	switch f.prevFrameStatus() {
	case prevFrameNone:
		return f.pushFirstFrame(p)

	case prevFrameDrain:
		return f.pushAfterDrain(p)

	case prevFrameOk:
		return f.pushAfterOk(p)
	}

	return Deny
}

func (f *Frame) pushFirstFrame(p *Packet) int {
	if f.First() != nil && f.First().Timestamp != p.Timestamp {
		return DenyTimestampInvalid
	}

	return f.Insert(p)
}

func (f *Frame) pushAfterDrain(p *Packet) int {
	if comapreTimestamp(p.Timestamp, f.prevFrameTimestamp()) <= 0 {
		return DenyTimestampInvalid
	}

	if f.First() != nil && p.Timestamp != f.First().Timestamp {
		return DenyTimestampInvalid
	}

	defer func() {
		if f.IsFull() && f.prevFrameSeq()+1 == f.First().Seq {
			f.done <- true
		}
	}()

	return f.Insert(p)
}

func (f *Frame) pushAfterOk(p *Packet) int {
	if comapreTimestamp(p.Timestamp, f.prevFrameTimestamp()) < 0 {
		return DenyTimestampInvalid
	}

	if f.First() != nil && p.Timestamp != f.First().Timestamp {
		return DenyTimestampInvalid
	}

	if f.IsFull() && f.prevFrameSeq()+1 == f.First().Seq {
		return DenyFrameFull
	}

	defer func() {
		if f.IsFull() && f.prevFrameSeq()+1 == f.First().Seq {
			f.done <- true
		}
	}()

	return f.Insert(p)
}

func (f *Frame) prevFrameStatus() int {
	if f.prevFrame == nil {
		return prevFrameNone
	}

	if f.prevFrame.IsFull() {
		return prevFrameOk
	}

	return prevFrameDrain
}

func (f *Frame) prevFrameSeq() uint16 {
	if f.prevFrame == nil {
		return 0
	}
	return f.prevFrame.Last().Seq
}

func (f *Frame) prevFrameTimestamp() uint32 {
	if f.prevFrame == nil {
		return 0
	}
	return f.prevFrame.First().Timestamp
}

func NewFrameWaitQueue() *FrameWaitQueue {
	return &FrameWaitQueue{
		mutex:  &sync.Mutex{},
		queue:  NewFrameQueue(),
		notify: make(chan int, 1),
	}
}

type FrameWaitQueue struct {
	mutex  *sync.Mutex
	queue  FrameQueue
	notify chan int
}

func (fw *FrameWaitQueue) Push(f *Frame) bool {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	empty := fw.queue.Empty()
	res := fw.queue.Push(f)
	if res && empty {
		fw.notify <- 1
	}
	return res
}

func (fw *FrameWaitQueue) Pop(ctx context.Context) (*Frame, error) {
	// first check
	fw.mutex.Lock()
	f := fw.queue.Pop()
	fw.mutex.Unlock()
	if f != nil {
		return f, nil
	}

	// second loop check
	for fw.queue.Empty() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-fw.notify:
			fw.mutex.Lock()
			f := fw.queue.Pop()
			fw.mutex.Unlock()
			if f != nil {
				return f, nil
			}
		}
	}

	// second double check
	fw.mutex.Lock()
	f = fw.queue.Pop()
	fw.mutex.Unlock()
	return f, nil
}

type FrameQueue interface {
	Empty() bool
	Push(*Frame) bool
	Peek() *Frame
	Pop() *Frame
}

func NewFrameQueue() FrameQueue {
	return &frameQueue{}
}

type frameNode struct {
	frame *Frame
	next  *frameNode
}

type frameQueue struct {
	head, tail *frameNode
	count      int
}

func (fq *frameQueue) Empty() bool {
	return fq.count == 0
}

func (fq *frameQueue) Push(f *Frame) (res bool) {
	res = true
	defer func() {
		if res {
			fq.count += 1
		}
	}()

	newNode := &frameNode{
		frame: f,
	}

	if fq.head == nil {
		fq.head = newNode
		fq.tail = newNode
		return
	}

	// <=
	hc := comapreTimestamp(f.Timestamp(), fq.head.frame.Timestamp())
	if hc == 0 {
		return
	} else if hc < 0 {
		fq.head.frame.prevFrame = f
		newNode.next = fq.head
		fq.head = newNode
		return
	}

	// >=
	tc := comapreTimestamp(f.Timestamp(), fq.tail.frame.Timestamp())
	if tc == 0 {
		return
	} else if tc > 0 {
		f.prevFrame = fq.tail.frame
		fq.tail.next = newNode
		fq.tail = newNode
		return
	}

	// ()
	prev := fq.head
	curr := prev.next
	for curr != nil {
		comp := comapreTimestamp(curr.frame.Timestamp(), f.Timestamp())
		if comp == 0 {
			break
		} else if comp > 0 {
			prev.next = newNode
			newNode.next = curr
			f.prevFrame = prev.frame
			curr.frame.prevFrame = f
			return
		}
		prev = curr
		curr = curr.next
	}
	res = false
	return
}

func (fq *frameQueue) Pop() *Frame {
	if fq.head != nil {
		f := fq.head.frame
		fq.count -= 1
		fq.head = fq.head.next
		return f
	}
	return nil
}

func (fq *frameQueue) Peek() *Frame {
	if fq.head != nil {
		return fq.head.frame
	}
	return nil
}
