package pool

import (
	"github.com/privacybydesign/gabi/big"
)

type PrimePool interface {
	Fetch(start, length uint) (*big.Int, error)
	StatsJSON() ([]byte, error)
}
