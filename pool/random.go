package pool

import (
	"encoding/json"
	"io"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
)

type randomPool struct {
	reader io.Reader
}

func (p *randomPool) StatsJSON() ([]byte, error) {
	type Stats struct {
		Name string
	}
	return json.Marshal(Stats{
		Name: "random",
	})
}

func NewRandomPool(r io.Reader) PrimePool {
	return &randomPool{
		reader: r,
	}
}

func (p *randomPool) Fetch(start, length uint) (*big.Int, error) {
	return common.RandomPrimeInRange(p.reader, start, length)
}
