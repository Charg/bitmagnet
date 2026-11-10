package processor

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/bitmagnet-io/bitmagnet/internal/database/dao"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/protocol"
	"github.com/bitmagnet-io/bitmagnet/internal/slice"
	"gorm.io/gorm/clause"
)

type persistPayload struct {
	torrentContents  []model.TorrentContent
	deleteIDs        []string
	deleteInfoHashes []protocol.ID
	addTags          map[protocol.ID]map[string]struct{}
}

func (c processor) persist(ctx context.Context, payload persistPayload) error {
	contentsMap := make(map[model.ContentRef]struct{}, len(payload.torrentContents))
	contentsPtr := make([]*model.Content, 0, len(payload.torrentContents))
	torrentContentsPtr := make([]*model.TorrentContent, 0, len(payload.torrentContents))
	torrentTagsPtr := make([]*model.TorrentTag, 0, len(payload.addTags))

	for _, tc := range payload.torrentContents {
		tcCopy := tc
		tcCopy.Torrent = model.Torrent{}

		if tcCopy.ContentID.Valid && tcCopy.Content.CreatedAt.IsZero() {
			contentRef := tcCopy.Content.Ref()
			if _, ok := contentsMap[contentRef]; !ok {
				contentsMap[contentRef] = struct{}{}
				contentCopy := tcCopy.Content
				contentsPtr = append(contentsPtr, &contentCopy)
			}
		}

		tcCopy.Content = model.Content{}
		torrentContentsPtr = append(torrentContentsPtr, &tcCopy)
	}

	for infoHash, tags := range payload.addTags {
		for tag := range tags {
			torrentTagsPtr = append(torrentTagsPtr, &model.TorrentTag{
				InfoHash: infoHash,
				Name:     tag,
			})
		}
	}

	if len(payload.deleteInfoHashes) > 0 {
		if blockErr := c.blockingManager.Block(ctx, payload.deleteInfoHashes, false); blockErr != nil {
			return blockErr
		}
	}

	return c.dao.Transaction(func(tx *dao.Query) error {
		if len(contentsPtr) > 0 {
			if createContentErr := tx.Content.WithContext(ctx).Clauses(
				clause.OnConflict{
					UpdateAll: true,
				}).CreateInBatches(contentsPtr, 100); createContentErr != nil {
				return createContentErr
			}
		}

		if len(payload.deleteIDs) > 0 {
			if _, deleteErr := tx.TorrentContent.WithContext(ctx).Where(
				c.dao.TorrentContent.ID.In(payload.deleteIDs...),
			).Delete(); deleteErr != nil {
				return deleteErr
			}
		}

		if len(torrentContentsPtr) > 0 {
			if createErr := tx.TorrentContent.WithContext(ctx).Clauses(
				clause.OnConflict{
					UpdateAll: true,
					// skip rewriting rows that haven't changed (the common case when reprocessing),
					// as every update
					// writes a new row version and new entries in each of the table's many indexes:
					Where: clause.Where{Exprs: []clause.Expression{torrentContentChanged}},
				},
			).CreateInBatches(torrentContentsPtr, 100); createErr != nil {
				return createErr
			}
		}

		if len(torrentTagsPtr) > 0 {
			if createErr := tx.TorrentTag.WithContext(ctx).Clauses(
				clause.OnConflict{
					DoNothing: true,
				},
			).CreateInBatches(torrentTagsPtr, 100); createErr != nil {
				return createErr
			}
		}

		if len(payload.deleteInfoHashes) > 0 {
			valuers := slice.Map(payload.deleteInfoHashes, func(infoHash protocol.ID) driver.Valuer {
				return infoHash
			})

			if _, deleteErr := tx.Torrent.WithContext(ctx).Where(
				c.dao.Torrent.InfoHash.In(valuers...),
			).Delete(); deleteErr != nil {
				return deleteErr
			}
		}

		return nil
	})
}

// torrentContentUpsertColumns are the columns rewritten when upserting torrent content
// (the columns that gorm's UpdateAll assigns, other than updated_at).
var torrentContentUpsertColumns = []string{
	"content_type",
	"content_source",
	"content_id",
	"languages",
	"episodes",
	"video_resolution",
	"video_source",
	"video_codec",
	"video_3d",
	"video_modifier",
	"release_group",
	"tsv",
	"seeders",
	"leechers",
	"size",
	"files_count",
}

// torrentContentChanged is an ON CONFLICT DO UPDATE condition that is true when any upserted column differs from the
// existing row.
var torrentContentChanged = func() clause.Expression {
	existing := make([]string, len(torrentContentUpsertColumns))
	excluded := make([]string, len(torrentContentUpsertColumns))

	for i, col := range torrentContentUpsertColumns {
		existing[i] = fmt.Sprintf("%q.%q", model.TableNameTorrentContent, col)
		excluded[i] = fmt.Sprintf("%q.%q", "excluded", col)
	}

	return clause.Expr{
		SQL: fmt.Sprintf(
			"(%s) IS DISTINCT FROM (%s)",
			strings.Join(existing, ", "),
			strings.Join(excluded, ", "),
		),
	}
}()
