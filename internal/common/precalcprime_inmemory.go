package common

import (
	"crypto/rand"
	"log"
	"sync"
	"time"

	"github.com/privacybydesign/gabi/big"
)

type inmemoryStorage struct {
	mu     sync.Mutex       // Mutex to guard index
	idx    uint64
	primes []big.Int        // Buffer with our new primes
	Size   uint64           // Maximum size of the buffer
	start  uint             // Minimum bit length for generating  primes
	length uint             // Bit length of generated primes
}

func NewInMemoryStorage(size uint64, start, length uint) *inmemoryStorage {
 	s := &inmemoryStorage{
		primes: make([]big.Int, size),
		Size:   size,
		start:  start,
		length: length,
	}

	// Separate goroutine to fill buffer. When full, it will back off for one second
	go func() {
		for {
			// Buffer full
			if (s.idx == s.Size - 1) {
				time.Sleep(1 * time.Second)
				continue
			}

			s.AddNewPrimeToBuffer()
		}
	}()

	return s
}

// Fetch a new prime directly from our in-memory buffer
func (s *inmemoryStorage) Fetch(_, _ uint) (*big.Int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Warn if depleted
	if s.idx == 0 {
		log.Printf("waning: the buffer has depleted (size: %d)\n", s.Size)
		return RandomPrimeInRange(rand.Reader, s.start, s.length)
	}

	p := s.primes[s.idx]
	s.idx--

	return &p, nil
}

// AddNewPrimeToBuffer will generate a new prime and add it to the buffer if not already full
func (s *inmemoryStorage) AddNewPrimeToBuffer() {
	if (s.idx == s.Size - 1) {
		return
	}

	p, err := RandomPrimeInRange(rand.Reader, s.start, s.length)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.primes[s.idx + 1] = *p
	s.idx++;
}
