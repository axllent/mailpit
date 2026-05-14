// Package shortuuid provides a simple way to generate short, unique, alphanumeric identifiers.
// The generated IDs are 22 characters long and consist of uppercase letters, lowercase letters, and digits.
package shortuuid

import (
	"encoding/binary"
	"math/bits"

	"github.com/google/uuid"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	length   = 22
	nDigits  = 10
	divisor  = 839299365868340224 // 62^10, max power of 62 that fits in uint64
)

// New returns a 22-character alphanumeric unique identifier.
func New() string {
	id := uuid.New()
	num := [2]uint64{
		binary.BigEndian.Uint64(id[8:]),
		binary.BigEndian.Uint64(id[:8]),
	}

	buf := make([]byte, length)
	var r uint64
	i := length - 1
	for num[1] > 0 || num[0] > 0 {
		num, r = quoRem64(num, divisor)
		for j := 0; j < nDigits && i >= 0; j++ {
			buf[i] = alphabet[r%62]
			r /= 62
			i--
		}
	}
	for ; i >= 0; i-- {
		buf[i] = alphabet[0]
	}

	return string(buf)
}

// quoRem64 divides a 128-bit number (represented as [lo, hi] uint64) by v,
// returning the quotient and remainder.
func quoRem64(u [2]uint64, v uint64) ([2]uint64, uint64) {
	var q [2]uint64
	var r uint64
	q[1], r = bits.Div64(0, u[1], v)
	q[0], r = bits.Div64(r, u[0], v)
	return q, r
}
