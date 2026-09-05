package main

import (
    "os"

    sdk "github.com/PlakarKorp/go-kloset-sdk"
    "b2Storage/storage"
)

func main() {
    sdk.EntrypointStorage(os.Args, storage.NewStore)
}
