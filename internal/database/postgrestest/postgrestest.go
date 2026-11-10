// Package postgrestest provides a migrated Postgres database for integration tests.
//
// Tests using it are skipped unless the BITMAGNET_TEST_POSTGRES_DSN environment variable is set to the DSN of a
// database that the tests may freely write to, e.g.
// BITMAGNET_TEST_POSTGRES_DSN="postgres://postgres@localhost:5432/bitmagnet_test?sslmode=disable".
package postgrestest

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/database/dao"
	migrationssql "github.com/bitmagnet-io/bitmagnet/migrations"
	goose "github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const EnvDSN = "BITMAGNET_TEST_POSTGRES_DSN"

var (
	migrateOnce sync.Once
	errMigrate  error
)

// Open returns a query for the test database, applying migrations once per test binary,
// or skips the test if no test database is configured.
func Open(t *testing.T) *dao.Query {
	t.Helper()

	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s is not set", EnvDSN)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open test database: %s", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get test database connection: %s", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	migrateOnce.Do(func() {
		goose.SetBaseFS(migrationssql.FS)

		if errMigrate = goose.SetDialect("postgres"); errMigrate != nil {
			return
		}

		errMigrate = goose.UpContext(context.Background(), sqlDB, ".")
	})

	if errMigrate != nil {
		t.Fatalf("failed to migrate test database: %s", errMigrate)
	}

	return dao.Use(db)
}
