package unifi

import (
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
)

type (
	Duration              int64
	TimeStamp             int64
	DurationMilliseconds  int64
	TimeStampMilliseconds int64
	MAC                   string
	IP                    string
	Number                int64
)

func (d Duration) String() string {
	return humanize.Time(time.Now().Add(-time.Second * time.Duration(d)))
}

func (t TimeStamp) String() string    { return humanize.Time(time.UnixMilli(int64(t))) }
func (t TimeStamp) ShortTime() string { return time.UnixMilli(int64(t)).Format("03:04:05PM") }

func (m MAC) String() string {
	if len(m) == 0 {
		return "00:00:00:00:00:00"
	}

	return string(m)
}

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

func (n *Number) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	var (
		s   string
		i   int64
		err error
	)

	if b[0] == '"' {
		if err = json.Unmarshal(b, &s); err != nil {
			return err
		}

		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			return err
		}

		*n = Number(i)

		return nil
	}

	if err = json.Unmarshal(b, &i); err != nil {
		return err
	}

	*n = Number(i)

	return nil
}

// nolint: gomnd
func formatBytesSize(size int64) string {
	if size <= 0 {
		return ""
	}

	return humanize.Bytes(uint64(size))
}

func firstNonEmpty(s ...string) string {
	for _, candidate := range s {
		if len(candidate) > 0 {
			return candidate
		}
	}

	return ""
}
