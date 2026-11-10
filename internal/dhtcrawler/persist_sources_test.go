package dhtcrawler

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/database/postgrestest"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistSourcesBatchSkipsDeletedTorrents checks that a scraped source whose torrent no longer exists
// doesn't fail the foreign key check for the rest of the batch.
func TestPersistSourcesBatchSkipsDeletedTorrents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	q := postgrestest.Open(t)
	c := &crawler{dao: q}

	existing := protocol.RandomNodeID()
	deleted := protocol.RandomNodeID()

	require.NoError(t, q.Torrent.WithContext(ctx).Create(&model.Torrent{
		InfoHash:    existing,
		Name:        "existing",
		FilesStatus: model.FilesStatusSingle,
	}))

	t.Cleanup(func() {
		_, _ = q.Torrent.WithContext(context.Background()).Where(
			q.Torrent.InfoHash.In(driver.Valuer(existing)),
		).Delete()
	})

	persisted, err := c.persistSourcesBatch(ctx, []*model.TorrentsTorrentSource{
		{Source: "dht", InfoHash: deleted, Seeders: model.NewNullUint(3), Leechers: model.NewNullUint(4)},
		{Source: "dht", InfoHash: existing, Seeders: model.NewNullUint(1), Leechers: model.NewNullUint(2)},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, persisted)

	src, err := q.TorrentsTorrentSource.WithContext(ctx).Where(
		q.TorrentsTorrentSource.InfoHash.Eq(existing),
	).First()
	require.NoError(t, err)
	assert.Equal(t, model.NewNullUint(1), src.Seeders)
	assert.Equal(t, model.NewNullUint(2), src.Leechers)

	// a second scrape updates the counts:
	persisted, err = c.persistSourcesBatch(ctx, []*model.TorrentsTorrentSource{
		{Source: "dht", InfoHash: existing, Seeders: model.NewNullUint(5), Leechers: model.NewNullUint(6)},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, persisted)

	src, err = q.TorrentsTorrentSource.WithContext(ctx).Where(
		q.TorrentsTorrentSource.InfoHash.Eq(existing),
	).First()
	require.NoError(t, err)
	assert.Equal(t, model.NewNullUint(5), src.Seeders)
}
