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
		// site-name prefixes
		{
			input:       "www.UIndex.org    -    Ted Lasso S02E07 Headspace 1080p ATVP WEB-DL DDP5 1 Atmos H 264-FLUX",
			contentType: tvShow,
			title:       "Ted Lasso",
			episodes:    model.Episodes{2: {7: {}}},
		},
		{
			input:       "www.Torrenting.com - The Rookie S07E06 The Gala 1080p AMZN WEB-DL DDP5 1 H 264-FLUX",
			contentType: tvShow,
			title:       "The Rookie",
			episodes:    model.Episodes{7: {6: {}}},
		},
		{
			input:       "www.SceneTime.com  - Saturday.Night.Live.S43E01.1080p.WEB.x264-TBS",
			contentType: tvShow,
			title:       "Saturday Night Live",
			episodes:    model.Episodes{43: {1: {}}},
		},
		{
			input:       "WWW.SCENETIME.COM - Saturday.Night.Live.S43E01.1080p.WEB.x264-TBS",
			contentType: tvShow,
			title:       "Saturday Night Live",
			episodes:    model.Episodes{43: {1: {}}},
		},
		{
			input:       "Torrenting.com - The Rookie S07E06 The Gala 1080p AMZN WEB-DL DDP5 1 H 264-FLUX",
			contentType: tvShow,
			title:       "The Rookie",
			episodes:    model.Episodes{7: {6: {}}},
		},
		{
			input:       "www.UIndex.org    -    The.Fantastic.Four.First.Steps.2025.1080p.WEB.h264-ETHEL",
			contentType: movie,
			title:       "The Fantastic Four First Steps",
			year:        2025,
		},
		{
			input:       "[Ex-torrenty.org]Nobody.Wants.This.S03.MULTi.1080p.WEB-DL.H264.DDP5.1.Atmos-K83",
			contentType: tvShow,
			title:       "Nobody Wants This",
			episodes:    model.Episodes{3: {}},
		},
		{
			input:       "【高清影视之家发布 www.HDBTHD.com】防弹武僧[中文字幕].Bulletproof.Monk.2003.BluRay.1080p.AAC.x264-DreamHD",
			contentType: movie,
			title:       "Bulletproof Monk",
			year:        2003,
		},
		{
			input:       "【高清剧集网发布 www.DDHDTV.com】芝加哥警署[第03集][无字片源].Chicago.P.D.S14.1080p.Peacock.WEB-DL.DDP.5.1.H.264-BlackTV",
			contentType: tvShow,
			title:       "Chicago P.D.",
			episodes:    model.Episodes{14: {}},
		},
		{
			input:       "The.Last.of.Us.S02E03.1080p.WEB.H264-SuccessfulCrab[EZTVx.to].mkv",
			contentType: tvShow,
			title:       "The Last of Us",
			episodes:    model.Episodes{2: {3: {}}},
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
