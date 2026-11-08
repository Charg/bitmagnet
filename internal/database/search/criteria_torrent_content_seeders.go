package search

import (
	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/maps"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"gorm.io/gen/field"
)

// TorrentContentMinSeedersCriteria matches torrent content with at least the given number of seeders,
// or an unknown number of seeders.
func TorrentContentMinSeedersCriteria(minSeeders uint) query.Criteria {
	return query.DaoCriteria{
		Conditions: func(ctx query.DBContext) ([]field.Expr, error) {
			q := ctx.Query()
			return []field.Expr{
				field.Or(
					q.TorrentContent.Seeders.IsNull(),
					q.TorrentContent.Seeders.Gte(model.NewNullUint(minSeeders)),
				),
			}, nil
		},
		Joins: maps.NewInsertMap(
			maps.MapEntry[string, struct{}]{Key: model.TableNameTorrentContent},
		),
	}
}
