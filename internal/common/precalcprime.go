// Copyright 2016 Maarten Everts. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"crypto/rand"

	"github.com/go-errors/errors"

	"github.com/privacybydesign/gabi/big"
)

type PrimeStorage interface {
	Fetch(start, length uint) (*big.Int, error)
}

// RandomPrecalcPrimeInRange returns a precalculated prime from a bolt storage
func RandomPrecalcPrimeInRange(storage PrimeStorage, start, length uint) (p *big.Int, err error) {
	if start < 2 {
		err = errors.New("randomPrimeInRange: prime size must be at least 2-bit")
		return
	}

	p, err = storage.Fetch(start, length)
	if err != nil {
		return RandomPrimeInRange(rand.Reader, start, length)
	}

	return p, err
}
