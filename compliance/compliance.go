package compliance

import (
	"encore.dev/storage/cache"
	"encore.dev/storage/sqldb"
)

var db = sqldb.NewDatabase("compliance", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

var complianceCache = cache.NewCluster("compliance", cache.ClusterConfig{
	EvictionPolicy: cache.AllKeysLRU,
})

//encore:service
type Service struct{}

func initService() (*Service, error) {
	return &Service{}, nil
}
