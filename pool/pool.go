package pool

import (
	"crypto/rand"

	"github.com/go-errors/errors"
	"github.com/privacybydesign/gabi/internal/common"

	"github.com/privacybydesign/gabi/big"
)

type PrimePool interface {
	Fetch(start, length uint) (*big.Int, error)
}

// RandomPrimeInRangeFromPool returns a precalculated prime from a bolt pool
func RandomPrimeInRangeFromPool(pool PrimePool, start, length uint) (p *big.Int, err error) {
	if start < 2 {
		err = errors.New("randomPrimeInRange: prime size must be at least 2-bit")
		return
	}

	p, err = pool.Fetch(start, length)
	if err != nil {
		return common.RandomPrimeInRange(rand.Reader, start, length)
	}

	return p, err
}
