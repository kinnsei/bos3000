package billing

import (
	"time"

	"encore.dev/storage/cache"
	"encore.dev/storage/sqldb"
)

var db = sqldb.NewDatabase("billing", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

var billingCache = cache.NewCluster("billing", cache.ClusterConfig{
	EvictionPolicy: cache.AllKeysLRU,
})

var concurrentSlots = cache.NewIntKeyspace[int64](billingCache, cache.KeyspaceConfig{
	KeyPattern:    "concurrent/:key",
	DefaultExpiry: cache.ExpireIn(24 * time.Hour),
})

//encore:service
type Service struct{}

func initService() (*Service, error) {
	return &Service{}, nil
}
