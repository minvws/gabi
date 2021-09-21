package pool

import (
	"crypto/rand"
	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
)

type randomPool struct {

}

func NewRandomPool() PrimePool {
	return &randomPool{}
}

func (p *randomPool) Fetch(start, length uint) (*big.Int, error) {
	return common.RandomPrimeInRange(rand.Reader, start, length)
}
