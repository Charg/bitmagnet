package parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripReleaseNameBanner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"www.UIndex.org    -    Ted Lasso S02E07", "Ted Lasso S02E07"},
		{"www.SceneTime.com  - Saturday.Night.Live.S43E01", "Saturday.Night.Live.S43E01"},
		{"WWW.SCENETIME.COM - Saturday.Night.Live.S43E01", "Saturday.Night.Live.S43E01"},
		{"Torrenting.com - The Rookie S07E06", "The Rookie S07E06"},
		{"[Ex-torrenty.org]Nobody.Wants.This.S03", "Nobody.Wants.This.S03"},
		{"[www.example.net] Some Movie 2020", "Some Movie 2020"},
		{
			"【高清影视之家发布 www.HDBTHD.com】防弹武僧[中文字幕].Bulletproof.Monk.2003.BluRay",
			"Bulletproof.Monk.2003.BluRay",
		},
		{
			"【高清剧集网发布 www.DDHDTV.com】芝加哥警署[第03集][无字片源].Chicago.P.D.S14.1080p",
			"Chicago.P.D.S14.1080p",
		},
		// unchanged:
		{"Dr.Who - The Movie 1996", "Dr.Who - The Movie 1996"},
		{"Mike.and.Me - 2020", "Mike.and.Me - 2020"},
		{"[SubsPlease] Sousou no Frieren - 12 (1080p)", "[SubsPlease] Sousou no Frieren - 12 (1080p)"},
		{"The.Last.of.Us.S02E03.1080p[EZTVx.to].mkv", "The.Last.of.Us.S02E03.1080p[EZTVx.to].mkv"},
		{"葬送のフリーレン 第05話", "葬送のフリーレン 第05話"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, stripReleaseNameBanner(test.input))
		})
	}
}
