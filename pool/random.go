package pool

import (
	"io"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
)

type randomPool struct {
	reader io.Reader
}

func NewRandomPool(r io.Reader) PrimePool {
	return &randomPool{
		reader: r,
	}
}

func (p *randomPool) Fetch(start, length uint) (*big.Int, error) {
	return common.RandomPrimeInRange(p.reader, start, length)
}
