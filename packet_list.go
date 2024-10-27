package rtp

const (
	AcceptOk = iota
	DenyPacketNil
	DenyTimestampInvalid
	DenyFrameFull
	DenyPacketDuplicated
	Deny
)

type Cursor interface {
	Next() *Packet
}

func NewPacketList() PacketList {
	return &packetList{}
}

type PacketList interface {
	NewCursor() Cursor
	IsFull() bool
	Count() int
	First() *Packet
	Last() *Packet
	Insert(*Packet) int
}

type Node struct {
	pkg  *Packet
	prev *Node
	next *Node
}

type cursor struct {
	curr *Node
}

func (c *cursor) Next() *Packet {
	if c.curr != nil {
		p := c.curr.pkg
		c.curr = c.curr.next
		return p
	}
	return nil
}

type packetList struct {
	head  *Node
	tail  *Node
	count int
}

func (list *packetList) NewCursor() Cursor {
	return &cursor{
		curr: list.head,
	}
}

func (list *packetList) Count() int {
	return list.count
}

func (list *packetList) First() *Packet {
	if list.head != nil {
		return list.head.pkg
	}
	return nil
}

func (list *packetList) Last() *Packet {
	if list.tail != nil {
		return list.tail.pkg
	}
	return nil
}

func (list *packetList) IsFull() bool {
	return list.tail != nil &&
		list.tail.pkg.Marker == 1 &&
		getCount(list.head.pkg.Seq, list.tail.pkg.Seq) == list.count
}

func getCount(start, end uint16) int {
	if start > end {
		return int(end) + int(65535-start) + 2
	}

	return int(end-start) + 1
}

func (list *packetList) Insert(p *Packet) int {
	ok := list.insert(p)
	if ok == AcceptOk {
		list.count += 1
	}
	return ok
}

// Binary search for the insertion point
func (list *packetList) findInsertPoint(p *Packet) *Node {
	low, high := list.head, list.tail
	for low != nil && high != nil && low != high && low.next != high {
		mid := low
		fast := low
		for fast != high && fast.next != high {
			mid = mid.next
			fast = fast.next.next
		}

		code := compareSequence(mid.pkg.Seq, p.Seq)
		if code == 0 {
			return mid
		} else if code == -1 {
			low = mid.next
		} else {
			high = mid.prev
		}
	}

	return low
}

func (list *packetList) insert(p *Packet) int {
	newNode := &Node{pkg: p}
	if list.head == nil {
		list.head = newNode
		list.tail = newNode
		return AcceptOk
	}

	tailComp := compareSequence(p.Seq, list.tail.pkg.Seq)
	headComp := compareSequence(p.Seq, list.head.pkg.Seq)

	// append tail: high priority
	if tailComp > 0 {
		list.tail.next = newNode
		newNode.prev = list.tail
		list.tail = newNode
		return AcceptOk
	}

	// append head
	if headComp < 0 {
		newNode.next = list.head
		list.head.prev = newNode
		list.head = newNode
		return AcceptOk
	}

	// duplicated on head or tail
	if tailComp == 0 || headComp == 0 {
		return DenyPacketDuplicated
	}

	// no slots in range [head:tail]
	if getCount(list.First().Seq, list.Last().Seq) == list.count {
		return DenyPacketDuplicated
	}

	insertPoint := list.findInsertPoint(p)
	if insertPoint == nil {
		newNode.next = list.head
		list.head.prev = newNode
		list.head = newNode
		return AcceptOk
	}

	if insertPoint.pkg.Seq == p.Seq {
		return DenyPacketDuplicated
	}

	comp := compareSequence(insertPoint.pkg.Seq, p.Seq)
	if comp > 0 {
		if insertPoint.prev == nil {
			insertPoint.prev = newNode
			newNode.next = insertPoint
			list.head = newNode
		} else {
			insertPoint.prev.next = newNode
			newNode.prev = insertPoint.prev
			newNode.next = insertPoint
			insertPoint.prev = newNode
		}
	} else {
		if insertPoint.next == nil {
			insertPoint.next = newNode
			newNode.prev = insertPoint
			list.tail = newNode
		} else {
			next := insertPoint.next
			insertPoint.next = newNode
			newNode.prev = insertPoint
			newNode.next = next
			next.prev = newNode
		}
	}

	return AcceptOk
}
