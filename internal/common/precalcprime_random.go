package common

import (
	"crypto/rand"

	"github.com/privacybydesign/gabi/big"
)

type randomStorage struct {
}

func NewRandomStorage() PrimeStorage {
	return &randomStorage{}
}

func (b *randomStorage) Fetch(start, length uint) (*big.Int, error) {
	return RandomPrimeInRange(rand.Reader, start, length)
}
