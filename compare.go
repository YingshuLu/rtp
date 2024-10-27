package rtp

// maxSamples = 90kHz
const maxSamples int64 = 90000

func comapreTimestamp(t1, t2 uint32) int {
	distance := int64(t1) - int64(t2)
	if distance == 0 {
		return 0
	} else if distance > 0 {
		if distance <= maxSamples {
			return 1
		} else {
			return -1
		}
	} else {
		if distance <= -maxSamples {
			return 1
		} else {
			return -1
		}
	}
}

func compareSequence(s1, s2 uint16) int {
	if s1 == s2 {
		return 0
	}
	distance := int(s1) - int(s2)
	if distance > 0 {
		// 65535 < 0
		if distance >= 60000 {
			return -1
		} else {
			return 1
		}
	} else {
		// 0 > 65535
		if distance <= -60000 {
			return 1
		} else {
			return -1
		}
	}
}
