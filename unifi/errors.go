package unifi

import "errors"

var (
	ErrNilSession           = errors.New("nil session")
	ErrUninitializedSession = errors.New("uninitialized session")
	ErrTooManyWriters       = errors.New("too many writers")
)
