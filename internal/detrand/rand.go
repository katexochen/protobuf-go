// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package detrand provides deterministically random functionality.
//
// The pseudo-randomness of these functions is seeded by the program binary
// itself and guarantees that the output does not change within a program,
// while ensuring that the output is unstable across different builds.
package detrand

import (
	"encoding/binary"
	"hash/fnv"
	"os"
)

// Bool returns a deterministically random boolean.
func Bool() bool {
	return binHash%2 == 0
}

// Intn returns a deterministically random integer within [0,n).
func Intn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	return int(binHash % uint64(n))
}

// binHash is a best-effort at an approximate hash of the Go binary.
var binHash = binaryHash()

func binaryHash() uint64 {
	// Open the Go binary.
	s, err := os.Executable()
	if err != nil {
		return 0
	}
	f, err := os.Open(s)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Hash the size and several samples of the Go binary.
	const numSamples = 8
	var buf [64]byte
	h := fnv.New64()
	fi, err := f.Stat()
	if err != nil {
		return 0
	}
	binary.LittleEndian.PutUint64(buf[:8], uint64(fi.Size()))
	h.Write(buf[:8])
	for i := int64(0); i < numSamples; i++ {
		if _, err := f.ReadAt(buf[:], i*fi.Size()/numSamples); err != nil {
			return 0
		}
		h.Write(buf[:])
	}
	return h.Sum64()
}