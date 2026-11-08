package adapter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/database/dao"
	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/database/search"
	"github.com/bitmagnet-io/bitmagnet/internal/lazy"
	"github.com/bitmagnet-io/bitmagnet/internal/torznab"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlRecorder is a gorm logger that records every statement gorm builds.
type sqlRecorder struct {
	mtx   sync.Mutex
	stmts []string
}

func (r *sqlRecorder) LogMode(logger.LogLevel) logger.Interface { return r }

func (*sqlRecorder) Info(context.Context, string, ...interface{}) {}

func (*sqlRecorder) Warn(context.Context, string, ...interface{}) {}

func (*sqlRecorder) Error(context.Context, string, ...interface{}) {}

func (r *sqlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()

	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.stmts = append(r.stmts, sql)
}

// searchRequestSQL builds the torrent content items query for a torznab request without executing it,
// returning the rendered SQL with variables interpolated.
func searchRequestSQL(t *testing.T, req torznab.SearchRequest) string {
	t.Helper()

	rec := &sqlRecorder{}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               rec,
	})
	require.NoError(t, err)

	s, err := search.New(search.Params{
		Query: lazy.New(func() (*dao.Query, error) {
			return dao.Use(db), nil
		}),
	}).Search.Get()
	require.NoError(t, err)

	reqOptions, err := searchRequestToQueryOptions(req)
	require.NoError(t, err)

	options := append(
		[]query.Option{search.TorrentContentDefaultOption(), query.WithTotalCount(false)},
		reqOptions...)
	_, err = s.TorrentContent(context.Background(), options...)
	require.NoError(t, err)

	rec.mtx.Lock()
	defer rec.mtx.Unlock()

	// the default strategy query is the one that doesn't use a CTE
	for _, stmt := range rec.stmts {
		if len(stmt) > 6 && stmt[:6] == "SELECT" {
			return stmt
		}
	}

	require.FailNow(t, "no select statement recorded", rec.stmts)

	return ""
}
