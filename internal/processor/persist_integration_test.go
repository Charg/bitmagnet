package processor

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

// TestPersistSkipsUnchangedTorrentContent checks that re-persisting identical torrent content doesn't write a new
// row version, while a changed column still updates the row.
func TestPersistSkipsUnchangedTorrentContent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	q := postgrestest.Open(t)
	p := processor{dao: q}

	infoHash := protocol.RandomNodeID()

	require.NoError(t, q.Torrent.WithContext(ctx).Create(&model.Torrent{
		InfoHash:    infoHash,
		Name:        "Some Movie 2020 1080p",
		FilesStatus: model.FilesStatusSingle,
	}))

	t.Cleanup(func() {
		_, _ = q.Torrent.WithContext(context.Background()).Where(
			q.Torrent.InfoHash.In(driver.Valuer(infoHash)),
		).Delete()
	})

	tc := model.TorrentContent{
		InfoHash:        infoHash,
		ContentType:     model.NewNullContentType(model.ContentTypeMovie),
		VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
		Seeders:         model.NewNullUint(1),
		Size:            100,
	}
	tc.UpdateTsv()

	rowVersion := func() (xmin string, updatedAt string) {
		t.Helper()

		require.NoError(t, q.TorrentContent.WithContext(ctx).UnderlyingDB().Raw(
			"SELECT xmin::text, updated_at::text FROM torrent_contents WHERE info_hash = ?", infoHash,
		).Row().Scan(&xmin, &updatedAt))

		return
	}

	require.NoError(t, p.persist(ctx, persistPayload{torrentContents: []model.TorrentContent{tc}}))

	xmin1, updatedAt1 := rowVersion()

	require.NoError(t, p.persist(ctx, persistPayload{torrentContents: []model.TorrentContent{tc}}))

	xmin2, updatedAt2 := rowVersion()
	assert.Equal(t, xmin1, xmin2, "unchanged content should not be rewritten")
	assert.Equal(t, updatedAt1, updatedAt2)

	tc.Seeders = model.NewNullUint(7)

	require.NoError(t, p.persist(ctx, persistPayload{torrentContents: []model.TorrentContent{tc}}))

	xmin3, _ := rowVersion()
	assert.NotEqual(t, xmin2, xmin3, "changed content should be updated")

	stored, err := q.TorrentContent.WithContext(ctx).Where(q.TorrentContent.InfoHash.Eq(infoHash)).First()
	require.NoError(t, err)
	assert.Equal(t, model.NewNullUint(7), stored.Seeders)
}
