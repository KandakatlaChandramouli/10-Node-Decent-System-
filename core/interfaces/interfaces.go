package interfaces

import "context"

type Storage interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close() error
}

type StateMachine interface {
	Apply(commitLog []byte) ([]byte, error)
	GetStateRoot() []byte
}

type Consensus interface {
	Propose(ctx context.Context, data []byte) error
	Commit() <-chan []byte
}

type Service interface {
	Name() string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() error
}

type DependentService interface {
	Service
	Dependencies() []string
}
