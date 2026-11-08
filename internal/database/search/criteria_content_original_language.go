package search

import (
	"database/sql/driver"

	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/maps"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"gorm.io/gen/field"
)

func ContentOriginalLanguageCriteria(languages ...model.Language) query.Criteria {
	valuers := make([]driver.Valuer, 0, len(languages))
	for _, lang := range languages {
		valuers = append(valuers, lang)
	}

	return query.DaoCriteria{
		Conditions: func(ctx query.DBContext) ([]field.Expr, error) {
			q := ctx.Query()
			return []field.Expr{
				q.Content.OriginalLanguage.In(valuers...),
			}, nil
		},
		Joins: maps.NewInsertMap(
			maps.MapEntry[string, struct{}]{Key: model.TableNameContent},
		),
	}
}
