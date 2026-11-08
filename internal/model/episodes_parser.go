package model

import (
	"strconv"
	"strings"

	"github.com/bitmagnet-io/bitmagnet/internal/keywords"
	"github.com/bitmagnet-io/bitmagnet/internal/regex"
	"github.com/hedhyw/rex/pkg/dialect"
	"github.com/hedhyw/rex/pkg/rex"
)

// rangeSeparatorToken matches the separator of a range, e.g. "-", " - " or "-E".
func rangeSeparatorToken(runes string) dialect.Token {
	return rex.Group.Define(
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Chars.Single('-'),
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Group.NonCaptured(
			rex.Chars.Runes(runes).Repeat().ZeroOrOne(),
			rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		).Repeat().ZeroOrOne(),
	).NonCaptured()
}

// seasonRangeSeparatorToken is like rangeSeparatorToken, except that a dash surrounded by whitespace
// and not followed by an "S" is excluded, as "S2 - 05" denotes an episode rather than a range of seasons.
var seasonRangeSeparatorToken = rex.Group.Composite(
	rex.Group.Define(
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Chars.Single('-'),
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Chars.Runes("sS"),
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
	).NonCaptured(),
	rex.Group.Define(
		rex.Chars.Single('-'),
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
	).NonCaptured(),
	rex.Group.Define(
		rex.Chars.Whitespace(),
		rex.Chars.Single('-'),
	).NonCaptured(),
).NonCaptured()

func rangeToken(runes string, separator dialect.Token, maxDigits int) dialect.Token {
	return rex.Group.Define(
		rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, maxDigits)),
		rex.Group.Composite(
			rex.Group.Define(
				separator,
				rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, maxDigits)),
			).NonCaptured(),
			rex.Group.Define(
				rex.Chars.Whitespace().Repeat().ZeroOrOne(),
				rex.Chars.Single(','),
				rex.Chars.Whitespace().Repeat().ZeroOrOne(),
				rex.Group.NonCaptured(
					rex.Chars.Runes(runes).Repeat().ZeroOrOne(),
					rex.Chars.Whitespace().Repeat().ZeroOrOne(),
				).Repeat().ZeroOrOne(),
				rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 2)),
				rex.Chars.Whitespace().Repeat().ZeroOrOne(),
			).NonCaptured().Repeat().OneOrMore(),
		).NonCaptured().Repeat().ZeroOrOne(),
	)
}

var seasonToken = rex.Group.Define(
	rex.Group.Composite(
		// the full "season" keyword may also be dot or underscore separated, e.g. "Season.3":
		rex.Group.Define(
			rex.Group.Composite(keywords.MustNewRexTokensFromKeywords("season")...).NonCaptured(),
			rex.Chars.Runes(" ._").Repeat().ZeroOrOne(),
		).NonCaptured(),
		rex.Group.Define(
			rex.Group.Composite(keywords.MustNewRexTokensFromKeywords("s")...).NonCaptured(),
			rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		).NonCaptured(),
	).NonCaptured(),
	// up to 3 digits for long-running shows, e.g. "S151E04":
	rangeToken("sS", seasonRangeSeparatorToken, 3),
	rex.Chars.Whitespace().Repeat().ZeroOrOne(),
).NonCaptured()

var episodeToken = rex.Group.Define(
	rex.Group.Composite(
		keywords.MustNewRexTokensFromKeywords("episode", "ep", "e")...,
	).NonCaptured(),
	rex.Chars.Whitespace().Repeat().ZeroOrOne(),
	rangeToken("eE", rangeSeparatorToken("eE"), 2),
).NonCaptured()

// dashEpisodeToken matches an episode number following a season, separated by a dash, as in "S2 - 05 (1080p)".
// The number may have a version suffix and must not be immediately followed by a letter or digit (e.g. "S2 - 1080p").
var dashEpisodeToken = rex.Group.Define(
	rex.Chars.Single('-'),
	rex.Chars.Whitespace().Repeat().OneOrMore(),
	rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 4)),
	rex.Group.NonCaptured(
		rex.Chars.Runes("vV"),
		rex.Chars.Digits().Repeat().OneOrMore(),
	).Repeat().ZeroOrOne(),
	rex.Group.Composite(
		regex.AnyNonWordChar(),
		rex.Chars.End(),
	).NonCaptured(),
).NonCaptured()

var episodesRegularTokens = rex.Group.Define(
	seasonToken,
	rex.Group.Composite(episodeToken, dashEpisodeToken).NonCaptured().Repeat().ZeroOrOne(),
).NonCaptured()

var episodesXFormatTokens = rex.Group.Define(
	rex.Group.Define(
		rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 2)),
		rex.Chars.Runes("xX"),
		rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 2)),
	).NonCaptured(),
	rex.Group.Define(
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Chars.Single('-'),
		rex.Chars.Whitespace().Repeat().ZeroOrOne(),
		rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 2)),
	).NonCaptured().Repeat().ZeroOrOne(),
).NonCaptured()

var EpisodesToken = rex.Group.Composite(
	episodesRegularTokens,
	episodesXFormatTokens,
)

var episodesRegex = rex.New(
	rex.Chars.Begin(),
	EpisodesToken,
	rex.Chars.End(),
).MustCompile()

func EpisodesMatchToEpisodes(match []string) Episodes {
	if len(match) < 13 {
		return nil
	}

	episodes := Episodes{}

	if match[1] != "" {
		// regular format
		seasonStart, _ := strconv.ParseInt(match[2], 10, 16)

		switch {
		case match[9] != "":
			// a single dash-separated episode
			episode, _ := strconv.ParseInt(match[9], 10, 16)
			episodes = episodes.AddEpisode(int(seasonStart), int(episode))
		case match[5] == "":
			// no episodes
			switch {
			case match[3] != "":
				// a season range
				seasonEnd, _ := strconv.ParseInt(match[3], 10, 16)
				for i := seasonStart; i <= seasonEnd; i++ {
					episodes = episodes.AddSeason(int(i))
				}
			case match[4] != "":
				// a list of seasons
				includedSeasons := strings.Split(match[1], ",")
				for _, season := range includedSeasons {
					seasonIndex, _ := strconv.ParseInt(strings.TrimLeft(strings.TrimSpace(season), "sS "), 10, 16)
					episodes = episodes.AddSeason(int(seasonIndex))
				}
			default:
				// or just a single season
				episodes = episodes.AddSeason(int(seasonStart))
			}
		default:
			// episodes
			episodeStart, _ := strconv.ParseInt(match[6], 10, 16)

			switch {
			case match[7] != "":
				// an episode range
				episodeEnd, _ := strconv.ParseInt(match[7], 10, 16)
				for i := episodeStart; i <= episodeEnd; i++ {
					episodes = episodes.AddEpisode(int(seasonStart), int(i))
				}
			case match[8] != "":
				// a list of episodes
				includedEpisodes := strings.Split(match[5], ",")
				for _, episode := range includedEpisodes {
					episodeIndex, _ := strconv.ParseInt(strings.TrimLeft(strings.TrimSpace(episode), "eE "), 10, 16)
					episodes = episodes.AddEpisode(int(seasonStart), int(episodeIndex))
				}
			default:
				// a single episode
				episodes = episodes.AddEpisode(int(seasonStart), int(episodeStart))
			}
		}
	} else {
		// x format
		season, _ := strconv.ParseInt(match[10], 10, 16)
		episodeStart, _ := strconv.ParseInt(match[11], 10, 16)
		episodeEnd := episodeStart

		if match[12] != "" {
			episodeEnd, _ = strconv.ParseInt(match[12], 10, 16)
		}

		for i := episodeStart; i <= episodeEnd; i++ {
			episodes = episodes.AddEpisode(int(season), int(i))
		}
	}

	return episodes
}

func ParseEpisodes(input string) Episodes {
	match := episodesRegex.FindStringSubmatch(input)
	if match == nil {
		return nil
	}

	// EpisodesMatchToEpisodes expects the match of EpisodesToken as its first element:
	return EpisodesMatchToEpisodes(match[1:])
}
