package storage

import (
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

type PebbleStorage struct {
	db *pebble.DB
}

func NewPebbleStorage(dirPath string) (*PebbleStorage, error) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	db, err := pebble.Open(dirPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db: %w", err)
	}

	return &PebbleStorage{db: db}, nil
}

func (p *PebbleStorage) Put(key, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *PebbleStorage) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	buf := make([]byte, len(val))
	copy(buf, val)
	return buf, nil
}

func (p *PebbleStorage) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *PebbleStorage) Close() error {
	return p.db.Close()
}
