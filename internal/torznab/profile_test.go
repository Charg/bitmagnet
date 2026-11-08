package torznab_test

import (
	"strings"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/torznab"
	"github.com/stretchr/testify/assert"
)

func TestCapsSupportedParams(t *testing.T) {
	t.Parallel()

	searching := torznab.ProfileDefault.Caps().Searching

	for _, testCase := range []struct {
		name     string
		search   torznab.CapsSearch
		expected []string
	}{
		{
			name:   "search",
			search: searching.Search,
			expected: []string{
				torznab.ParamQuery, torznab.ParamIMDBID, torznab.ParamTMDBID, torznab.ParamTVDBID,
			},
		},
		{
			name:   "tv-search",
			search: searching.TvSearch,
			expected: []string{
				torznab.ParamQuery, torznab.ParamIMDBID, torznab.ParamTMDBID, torznab.ParamTVDBID,
				torznab.ParamSeason, torznab.ParamEpisode,
			},
		},
		{
			name:     "movie-search",
			search:   searching.MovieSearch,
			expected: []string{torznab.ParamQuery, torznab.ParamIMDBID, torznab.ParamTMDBID},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "yes", testCase.search.Available)
			assert.ElementsMatch(t, testCase.expected, strings.Split(testCase.search.SupportedParams, ","))
		})
	}
}
