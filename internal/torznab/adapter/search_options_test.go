package adapter

import (
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/torznab"
	"github.com/stretchr/testify/assert"
)

const (
	sqlIMDBMovie = `"content_attributes"."content_type" = 'movie' AND "content_attributes"."content_type" = "content"."type"`
	sqlIMDBTV    = `"content_attributes"."content_type" = 'tv_show' AND "content_attributes"."content_type" = "content"."type"`
	sqlTMDBMovie = `"content"."type" = 'movie' AND "content"."source" = 'tmdb' AND "content"."id" = '496243'`
	sqlTMDBTV    = `"content"."type" = 'tv_show' AND "content"."source" = 'tmdb' AND "content"."id" = '250923'`
)

func TestSearchRequestToQueryOptions(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		request     torznab.SearchRequest
		contains    []string
		notContains []string
	}{
		{
			name: "tv search with imdb and tmdb ids matches either id",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionTV,
				IMDBID: model.NewNullString("26933824"),
				TMDBID: model.NewNullString("250923"),
			},
			contains: []string{
				sqlIMDBTV,
				`"content_attributes"."source" = 'imdb' AND "content_attributes"."value" = 'tt26933824') OR (` +
					sqlTMDBTV,
			},
		},
		{
			name: "movie search with imdb and tmdb ids matches either id",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionMovie,
				IMDBID: model.NewNullString("tt6751668"),
				TMDBID: model.NewNullString("496243"),
			},
			contains: []string{
				sqlIMDBMovie,
				`"content_attributes"."value" = 'tt6751668') OR (` + sqlTMDBMovie,
			},
		},
		{
			name: "tv search with imdb id restricts to tv shows",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionTV,
				IMDBID: model.NewNullString("26933824"),
			},
			contains:    []string{sqlIMDBTV},
			notContains: []string{"'movie'"},
		},
		{
			name: "movie search with tmdb id restricts to movies",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionMovie,
				TMDBID: model.NewNullString("496243"),
			},
			contains:    []string{sqlTMDBMovie},
			notContains: []string{"'tv_show'"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			req := testCase.request
			if req.Profile.ID == "" {
				req.Profile = torznab.ProfileDefault
			}

			sql := searchRequestSQL(t, req)

			for _, s := range testCase.contains {
				assert.Contains(t, sql, s)
			}

			for _, s := range testCase.notContains {
				assert.NotContains(t, sql, s)
			}
		})
	}
}
