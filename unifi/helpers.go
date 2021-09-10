package unifi

import (
	"fmt"
	"math"
	"net"
	"time"
)

type (
	Duration              int64
	TimeStamp             int64
	DurationMilliseconds  int64
	TimeStampMilliseconds int64
	MAC                   string
	IP                    string
)

func (d Duration) String() string { return (time.Second * time.Duration(d)).String() }

func (lhs IP) Less(rhs IP) bool {
	if len(rhs) == 0 {
		return false
	}

	if len(lhs) == 0 {
		return true
	}

	lip := net.ParseIP(string(lhs))
	rip := net.ParseIP(string(rhs))

	if len(rip) < len(lip) {
		return false
	}

	if len(lip) < len(rip) {
		return true
	}

	for ix := 0; ix < len(lip); ix++ {
		if rip[ix] < lip[ix] {
			return false
		}

		if lip[ix] < rip[ix] {
			return true
		}
	}

	return false
}

// nolint: gochecknoglobals
var (
	suffixes = []string{"B", "KB", "MB", "GB", "TB"}
)

// nolint: gomnd
func round(val, roundOn float64, places int) float64 {
	var round float64

	pow := math.Pow(10, float64(places))
	digit := pow * val

	if _, div := math.Modf(digit); div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}

	return round / pow
}

// nolint: gomnd
func formatBytesSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}

	base := math.Log(float64(size)) / math.Log(1024)
	rounded := round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	suffix := suffixes[int(math.Floor(base))]

	return fmt.Sprintf("%.2f %s", rounded, suffix)
}

func firstNonEmpty(s ...string) string {
	for _, candidate := range s {
		if len(candidate) > 0 {
			return candidate
		}
	}

	return ""
}
