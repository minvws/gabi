package common

import (
	"fmt"

	"github.com/privacybydesign/gabi/big"
	bolt "go.etcd.io/bbolt"
)

type boltStorage struct {
	client *bolt.DB
}

// BucketName is where the primes in start/length are stored (sprintf'ed)
const BucketName = "primes_%d_%d"

// BoltDBFile is the filename of the boltDB storage
const BoltDBFile = "primes.db"

var BoltStorage = New()

func New() PrimeStorage {
	db, err := bolt.Open(BoltDBFile, 0600, nil)
	if err != nil {
		return nil
	}

	return &boltStorage{
		client: db,
	}
}

func (b *boltStorage) Fetch(start, length uint) (*big.Int, error) {
	var bi = &big.Int{}

	err := b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmt.Sprintf(BucketName, start, length)))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		k, _ := c.First()

		bi.SetString(string(k[:]), 10)

		return bucket.Delete(k)
	})

	return bi, err
}
