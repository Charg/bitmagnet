package processor

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TestTorrentContentChangedCoversUpdatedColumns checks that the condition guarding the torrent content upsert
// compares exactly the columns that gorm's UpdateAll assigns (other than updated_at); if a column were missing
// from the comparison, a change to only that column would be silently dropped.
func TestTorrentContentChangedCoversUpdatedColumns(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	tx := db.Clauses(clause.OnConflict{
		UpdateAll: true,
		Where:     clause.Where{Exprs: []clause.Expression{torrentContentChanged}},
	}).Create(&[]model.TorrentContent{{
		InfoHash:    protocol.ID{1},
		ContentType: model.NewNullContentType(model.ContentTypeMovie),
	}})
	require.NoError(t, tx.Error)

	sql := tx.Statement.SQL.String()

	parts := regexp.MustCompile(`DO UPDATE SET (.+) WHERE (.+?)(?: RETURNING .*)?$`).FindStringSubmatch(sql)
	require.Len(t, parts, 3, sql)

	var assigned []string

	for _, m := range regexp.MustCompile(`"(\w+)"=`).FindAllStringSubmatch(parts[1], -1) {
		if m[1] != "updated_at" {
			assigned = append(assigned, m[1])
		}
	}

	compared := append([]string{}, torrentContentUpsertColumns...)

	sort.Strings(assigned)
	sort.Strings(compared)
	assert.Equal(t, assigned, compared)

	assert.True(t, strings.HasPrefix(parts[2], `("torrent_contents"."content_type", `), parts[2])
	assert.Contains(t, parts[2], `) IS DISTINCT FROM ("excluded"."content_type", `)
}
