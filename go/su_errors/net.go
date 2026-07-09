package su_errors

import "errors"

var (
	ErrIncompletePacket = errors.New("incomplete packet")
	ErrInvalidPacket    = errors.New("invalid packet")
)
