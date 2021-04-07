package unifi

import (
	"fmt"
	"math"
)

// nolint: gochecknoglobals,unused
var (
	suffixes = []string{"B", "KB", "MB", "GB", "TB"}
)

// nolint: unused,gomnd
func round(val float64, roundOn float64, places int) float64 {
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

// nolint: deadcode,unused,gomnd
func formatBytesSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}

	base := math.Log(float64(size)) / math.Log(1024)
	rounded := round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	suffix := suffixes[int(math.Floor(base))]

	return fmt.Sprintf("%.2f %s", rounded, suffix)
}

// nolint: deadcode,unused
func firstNonEmpty(s ...string) string {
	for _, candidate := range s {
		if len(candidate) > 0 {
			return candidate
		}
	}

	return ""
}
