package parsers

import (
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/classifier/classification"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVideoContent(t *testing.T) {
	t.Parallel()

	movie := model.NewNullContentType(model.ContentTypeMovie)
	tvShow := model.NewNullContentType(model.ContentTypeTvShow)

	tests := []struct {
		input       string
		contentType model.NullContentType
		title       string
		year        model.Year
		episodes    model.Episodes
	}{
		// regression baseline
		{
			input:       "Frieren.Beyond.Journeys.End.S01E12.1080p.WEB.H264-GROUP",
			contentType: tvShow,
			title:       "Frieren Beyond Journeys End",
			episodes:    model.Episodes{1: {12: {}}},
		},
		{
			input:       "Shogun.2024.S01.COMPLETE.2160p.WEB-DL.DDP5.1.H.265-GROUP",
			contentType: tvShow,
			title:       "Shogun",
			year:        2024,
			episodes:    model.Episodes{1: {}},
		},
		{
			input:       "Dune.Part.Two.2024.2160p.UHD.BluRay.x265-GROUP",
			contentType: movie,
			title:       "Dune Part Two",
			year:        2024,
		},
		{
			input:       "[EMBER] Chainsaw Man (2022) (Season 1) [1080p]",
			contentType: tvShow,
			title:       "Chainsaw Man",
			year:        2022,
			episodes:    model.Episodes{1: {}},
		},
		{
			input:       "[Judas] Jujutsu Kaisen (Season 2) [1080p][HEVC x265 10bit][Multi-Subs] (Batch)",
			contentType: tvShow,
			title:       "Jujutsu Kaisen",
			episodes:    model.Episodes{2: {}},
		},
		// genuine season ranges
		{
			input:       "Friends.S01-S05.1080p.BluRay.x264-GROUP",
			contentType: tvShow,
			title:       "Friends",
			episodes:    model.Episodes{1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
		},
		{
			input:       "Friends S01-05 1080p BluRay x264-GROUP",
			contentType: tvShow,
			title:       "Friends",
			episodes:    model.Episodes{1: {}, 2: {}, 3: {}, 4: {}, 5: {}},
		},
		{
			input:       "Friends Season 1-3 1080p BluRay x264-GROUP",
			contentType: tvShow,
			title:       "Friends",
			episodes:    model.Episodes{1: {}, 2: {}, 3: {}},
		},
		{
			input:       "Pokemon.S07-S09.DVDRip.x264-GROUP",
			contentType: tvShow,
			title:       "Pokemon",
			episodes:    model.Episodes{7: {}, 8: {}, 9: {}},
		},
		// "S2 - 05" is season 2 episode 5, not a season range
		{
			input:       "[SubsPlease] Sousou no Frieren S2 - 05 (720p) [6FD1E83B].mkv",
			contentType: tvShow,
			title:       "Sousou no Frieren",
			episodes:    model.Episodes{2: {5: {}}},
		},
		{
			input:       "[SubsPlease] Shin no Nakama S2 - 10 (1080p) [C46A4C07].mkv",
			contentType: tvShow,
			title:       "Shin no Nakama",
			episodes:    model.Episodes{2: {10: {}}},
		},
		{
			input:       "[Erai-raws] Ore dake Level Up na Ken Season 2 - 03 [1080p CR WEBRip HEVC EAC3][MultiSub][256751B7].mkv",
			contentType: tvShow,
			title:       "Ore dake Level Up na Ken",
			episodes:    model.Episodes{2: {3: {}}},
		},
		{
			input:       "[ASW] Jibaku Shounen Hanako-kun S2 - 03 [1080p HEVC][C9D060F9].mkv",
			contentType: tvShow,
			title:       "Jibaku Shounen Hanako-kun",
			episodes:    model.Episodes{2: {3: {}}},
		},
		// dotted "Season.N"
		{
			input:       "The.Bear.Season.3.1080p.HULU.WEB-DL.DDP5.1.H.264-GROUP",
			contentType: tvShow,
			title:       "The Bear",
			episodes:    model.Episodes{3: {}},
		},
		{
			input:       "The Bear (Season 2) 1080p",
			contentType: tvShow,
			title:       "The Bear",
			episodes:    model.Episodes{2: {}},
		},
		// episode version suffix
		{
			input:       "Boku no Hero Academia - S07E06v2 - 1080p WEB x264 -NanDesuKa (CR).mkv",
			contentType: tvShow,
			title:       "Boku no Hero Academia",
			episodes:    model.Episodes{7: {6: {}}},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			attrs, err := ParseVideoContent(model.Torrent{Name: test.input}, classification.Result{})
			require.NoError(t, err)
			assert.Equal(t, test.contentType, attrs.ContentType, "content type")
			assert.Equal(t, test.title, attrs.BaseTitle.String, "title")
			assert.Equal(t, test.year, attrs.Date.Year, "year")
			assert.Equal(t, test.episodes, attrs.Episodes, "episodes")
		})
	}
}
