package storage

import (
    "context"
    "fmt"

    "github.com/PlakarKorp/kloset/connectors/storage"
)

type Store struct {
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
    return nil, fmt.Errorf("hello")
}
