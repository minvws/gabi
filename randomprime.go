package gabi

import (
	"io"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
)

func RandomPrimeInRange(rand io.Reader, start, length uint) (p *big.Int, err error) {
	return common.RandomPrimeInRange(rand, start, length)
}

