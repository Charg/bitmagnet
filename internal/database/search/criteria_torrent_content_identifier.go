package search

import (
	"strings"

	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/maps"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"gorm.io/gen"
	"gorm.io/gorm"
)

// TorrentContentIdentifierCriteria matches torrent content whose content has any of the given canonical
// identifiers (the content source and ID, e.g. tmdb) or alternative identifiers (content attributes, e.g. imdb).
//
// The identifiers are resolved to content refs in a subquery, so the torrent content is found through its
// content ref index. An OR of per-identifier conditions on the joined content table can't use an index, and
// Postgres then filters every torrent content row of the type.
func TorrentContentIdentifierCriteria(canonical, alternative []model.ContentRef) query.Criteria {
	canonicalGroups := groupContentRefs(canonical)
	alternativeGroups := groupContentRefs(alternative)

	return query.GenCriteria(func(ctx query.DBContext) (query.Criteria, error) {
		q := ctx.Query()
		subQueries := make([]interface{}, 0, len(canonicalGroups)+len(alternativeGroups))

		for _, g := range canonicalGroups {
			conds := make([]gen.Condition, 0, 3)
			if !g.contentType.IsNil() {
				conds = append(conds, q.Content.Type.Eq(g.contentType.String()))
			}

			conds = append(conds, q.Content.Source.Eq(g.source), q.Content.ID.In(g.ids...))
			subQueries = append(subQueries, q.Content.
				Select(q.Content.Type, q.Content.Source, q.Content.ID).
				Where(conds...).
				UnderlyingDB())
		}

		for _, g := range alternativeGroups {
			conds := make([]gen.Condition, 0, 3)
			if !g.contentType.IsNil() {
				conds = append(conds, q.ContentAttribute.ContentType.Eq(g.contentType))
			}

			conds = append(conds,
				q.ContentAttribute.Source.Eq(g.source),
				q.ContentAttribute.Value.In(g.ids...),
			)
			subQueries = append(subQueries, q.ContentAttribute.
				Select(
					q.ContentAttribute.ContentType,
					q.ContentAttribute.ContentSource,
					q.ContentAttribute.ContentID,
				).
				Where(conds...).
				UnderlyingDB())
		}

		if len(subQueries) == 0 {
			return query.RawCriteria{Query: "FALSE"}, nil
		}

		return query.RawCriteria{
			Query: q.TorrentContent.UnderlyingDB().
				Session(&gorm.Session{NewDB: true}).
				Where(
					`("torrent_contents"."content_type", "torrent_contents"."content_source", `+
						`"torrent_contents"."content_id") IN (`+
						strings.Repeat("? UNION ALL ", len(subQueries)-1)+`?)`,
					subQueries...,
				),
			Joins: maps.NewInsertMap(
				maps.MapEntry[string, struct{}]{Key: model.TableNameTorrentContent},
			),
		}, nil
	})
}

type contentRefGroup struct {
	contentType model.ContentType
	source      string
	ids         []string
}

// groupContentRefs groups refs by content type and source, in order of first appearance so the SQL is stable.
func groupContentRefs(refs []model.ContentRef) []contentRefGroup {
	var groups []contentRefGroup

	index := make(map[[2]string]int)
	seen := make(map[model.ContentRef]struct{})

	for _, ref := range refs {
		if _, ok := seen[ref]; ok {
			continue
		}

		seen[ref] = struct{}{}

		key := [2]string{ref.Type.String(), ref.Source}

		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i

			groups = append(groups, contentRefGroup{contentType: ref.Type, source: ref.Source})
		}

		groups[i].ids = append(groups[i].ids, ref.ID)
	}

	return groups
}
