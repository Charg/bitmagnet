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
	sqlIMDBAny   = `EXISTS (SELECT * FROM "content_attributes" WHERE "content_attributes"."content_type" = "content"."type"`
	sqlTMDBMovie = `"content"."type" = 'movie' AND "content"."source" = 'tmdb' AND "content"."id" = '496243'`
	sqlTMDBTV    = `"content"."type" = 'tv_show' AND "content"."source" = 'tmdb' AND "content"."id" = '250923'`
	sqlTMDBAny   = `("content"."source" = 'tmdb' AND "content"."id" = '496243')`
	sqlAnime     = `"content"."original_language" = 'ja'`
	sqlAnimation = `"content_collections_content"."content_collection_type" = 'genre' AND ` +
		`"content_collections_content"."content_collection_source" = 'tmdb' AND ` +
		`"content_collections_content"."content_collection_id" = '16'`
	sqlTVDBTV = `"content_attributes"."source" = 'tvdb' AND "content_attributes"."value" = '421069'`
	// MinSeeders
	sqlMinSeeders = `("torrent_contents"."seeders" IS NULL OR "torrent_contents"."seeders" >= 5)`
	// ordering
	sqlOrderNewest  = `SELECT *, "torrent_contents"."published_at" AS _order_0 FROM`
	sqlOrderSeeders = `SELECT *, coalesce(torrent_contents.seeders, -1) AS _order_0, ` +
		`"torrent_contents"."published_at" AS _order_1, "torrent_contents"."info_hash" AS _order_2 FROM`
	sqlOrderSeedersBy = `ORDER BY "_order_0" DESC,"_order_1" DESC,"_order_2" DESC`
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
			name: "generic search with imdb id does not restrict content type",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionSearch,
				IMDBID: model.NewNullString("6751668"),
			},
			contains:    []string{sqlIMDBAny, `"content_attributes"."value" = 'tt6751668'`},
			notContains: []string{"'movie'", "'tv_show'"},
		},
		{
			name: "generic search with tmdb id does not restrict content type",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionSearch,
				TMDBID: model.NewNullString("496243"),
			},
			contains:    []string{sqlTMDBAny},
			notContains: []string{"'movie'", "'tv_show'"},
		},
		{
			name: "tv search with tvdb id",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionTV,
				TVDBID: model.NewNullString("421069"),
			},
			contains:    []string{sqlIMDBTV, sqlTVDBTV},
			notContains: []string{"'movie'"},
		},
		{
			name: "tv search with imdb, tmdb and tvdb ids matches any id",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionTV,
				IMDBID: model.NewNullString("26933824"),
				TMDBID: model.NewNullString("250923"),
				TVDBID: model.NewNullString("421069"),
			},
			contains: []string{
				`"content_attributes"."value" = 'tt26933824') OR (` + sqlTMDBTV + `) OR EXISTS (`,
				sqlTVDBTV,
			},
		},
		{
			name: "tv search with anime category",
			request: torznab.SearchRequest{
				Type: torznab.FunctionTV,
				Cats: []int{torznab.CategoryTVAnime.ID},
			},
			contains: []string{`"torrent_contents"."content_type" = 'tv_show'`, sqlAnime, sqlAnimation},
		},
		{
			name: "generic search with anime category restricts to tv shows",
			request: torznab.SearchRequest{
				Type: torznab.FunctionSearch,
				Cats: []int{torznab.CategoryTVAnime.ID},
			},
			contains: []string{`"torrent_contents"."content_type" = 'tv_show'`, sqlAnime, sqlAnimation},
		},
		{
			name: "tv search with tv and anime categories matches either",
			request: torznab.SearchRequest{
				Type: torznab.FunctionTV,
				Cats: []int{torznab.CategoryTV.ID, torznab.CategoryTVAnime.ID},
			},
			contains: []string{`"torrent_contents"."content_type" = 'tv_show' OR (`, sqlAnime, sqlAnimation},
		},
		{
			name: "tv search with hd category does not filter anime",
			request: torznab.SearchRequest{
				Type: torznab.FunctionTV,
				Cats: []int{torznab.CategoryTVHD.ID},
			},
			notContains: []string{sqlAnime},
		},
		{
			name: "min seeders excludes torrents with known seeders below the threshold",
			request: torznab.SearchRequest{
				Profile: profileWithMinSeeders(5),
				Type:    torznab.FunctionSearch,
			},
			contains: []string{sqlMinSeeders},
		},
		{
			name: "min seeders disabled by default",
			request: torznab.SearchRequest{
				Type: torznab.FunctionSearch,
			},
			notContains: []string{"seeders"},
		},
		{
			name: "rss orders by newest",
			request: torznab.SearchRequest{
				Type: torznab.FunctionTV,
				Cats: []int{torznab.CategoryTV.ID},
			},
			contains:    []string{sqlOrderNewest},
			notContains: []string{"seeders"},
		},
		{
			name: "tv id search without query orders by seeders",
			request: torznab.SearchRequest{
				Type:    torznab.FunctionTV,
				IMDBID:  model.NewNullString("26933824"),
				TMDBID:  model.NewNullString("250923"),
				Season:  model.NewNullInt(3),
				Episode: model.NewNullInt(1),
			},
			contains: []string{sqlOrderSeeders, sqlOrderSeedersBy},
		},
		{
			name: "movie id search without query orders by seeders",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionMovie,
				IMDBID: model.NewNullString("6751668"),
			},
			contains: []string{sqlOrderSeeders, sqlOrderSeedersBy},
		},
		{
			name: "tv season search without query or ids keeps newest first",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionTV,
				Season: model.NewNullInt(1),
			},
			contains:    []string{sqlOrderNewest},
			notContains: []string{"seeders"},
		},
		{
			name: "id search with query orders by relevance",
			request: torznab.SearchRequest{
				Type:   torznab.FunctionMovie,
				Query:  "foo",
				IMDBID: model.NewNullString("6751668"),
			},
			contains:    []string{"SELECT *, ts_rank_cd(torrent_contents.tsv, 'foo'::tsquery) AS _order_0 FROM"},
			notContains: []string{"seeders"},
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

func profileWithMinSeeders(minSeeders uint) torznab.Profile {
	profile := torznab.ProfileDefault
	profile.MinSeeders = minSeeders

	return profile
}
