package core

import (
	"context"
)

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comics, error)
	SearchIndex(ctx context.Context, phrase string, limit int) ([]Comics, error)
	BuildIndex(ctx context.Context) error
}

type DB interface {
	Search(ctx context.Context, keyword string) ([]int, error)
	Get(ctx context.Context, ID int) (Comics, error)
	LastID(ctx context.Context) (int, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type EventSubscriber interface {
	SubscribeDBUpdateEvent(ctx context.Context) (<-chan struct{}, error)
	SubscribeDBDropEvent(ctx context.Context) (<-chan struct{}, error)
	Close() error
}
