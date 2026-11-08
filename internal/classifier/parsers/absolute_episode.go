package parsers

import (
	"strconv"

	"github.com/bitmagnet-io/bitmagnet/internal/classifier/classification"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/regex"
	"github.com/hedhyw/rex/pkg/rex"
)

// absoluteEpisodeRegex matches absolute-numbered anime releases, e.g. "[SubsPlease] Sousou no Frieren - 12 (1080p)":
// a leading group tag, the title, then " - NN" with an optional version suffix,
// followed by an opening bracket, a file extension or the end of the name (excluding dates such as "- 2023-04-09").
var absoluteEpisodeRegex = rex.New(
	rex.Chars.Begin(),
	rex.Chars.Single('['),
	rex.Common.NotClass(rex.Chars.Single(']')).Repeat().OneOrMore(),
	rex.Chars.Single(']'),
	rex.Chars.Whitespace().Repeat().ZeroOrMore(),
	rex.Group.Define(rex.Chars.Any().Repeat().OneOrMore()),
	rex.Chars.Whitespace().Repeat().OneOrMore(),
	rex.Chars.Single('-'),
	rex.Chars.Whitespace().Repeat().OneOrMore(),
	rex.Group.Define(rex.Chars.Digits().Repeat().Between(1, 4)),
	rex.Group.NonCaptured(
		rex.Chars.Runes("vV"),
		rex.Chars.Digits().Repeat().OneOrMore(),
	).Repeat().ZeroOrOne(),
	rex.Group.Composite(
		rex.Group.Define(
			rex.Chars.Whitespace().Repeat().ZeroOrMore(),
			rex.Chars.Runes("(["),
		).NonCaptured(),
		rex.Group.Define(
			rex.Chars.Whitespace().Repeat().ZeroOrMore(),
			rex.Chars.End(),
		).NonCaptured(),
		rex.Group.Define(
			rex.Chars.Single('.'),
			regex.AnyWordChar().Repeat().Between(2, 4),
			rex.Chars.End(),
		).NonCaptured(),
	).NonCaptured(),
).MustCompile()

// embeddedEpisodesRegex finds explicit episodes anywhere in a string, e.g. "(S02E06)".
var embeddedEpisodesRegex = rex.New(
	rex.Group.Composite(
		rex.Chars.Begin(),
		regex.AnyNonWordChar(),
	).NonCaptured(),
	model.EpisodesToken,
).MustCompile()

// parseAbsoluteEpisode parses absolute-numbered anime releases.
// An absolute episode number can't be mapped to a season without external data,
// so the returned episodes are only populated if the name also contains explicit episodes, e.g. "- 34 (S02E06)".
func parseAbsoluteEpisode(input string) (string, model.Year, model.Episodes, string, error) {
	match := absoluteEpisodeRegex.FindStringSubmatchIndex(input)
	if match == nil {
		return "", 0, nil, "", classification.ErrUnmatched
	}

	rawTitle := input[match[2]:match[3]]
	number := input[match[4]:match[5]]
	rest := input[match[5]:]

	// a 4 digit number is more likely a year than an episode:
	if n, _ := strconv.Atoi(number); len(number) == 4 && n >= 1900 && n < 2100 {
		return "", 0, nil, "", classification.ErrUnmatched
	}

	// leave names such as "Title S2 - 05" to the regular episodes parser:
	if titleEpisodesRegex.MatchString(rawTitle) {
		return "", 0, nil, "", classification.ErrUnmatched
	}

	title, year, _, err := parseTitleYear(rawTitle)
	if err != nil {
		title = cleanTitle(rawTitle)
	}

	if title == "" {
		return "", 0, nil, "", classification.ErrUnmatched
	}

	var episodes model.Episodes
	if m := embeddedEpisodesRegex.FindStringSubmatch(rest); m != nil {
		episodes = model.EpisodesMatchToEpisodes(m[1:])
	}

	return title, year, episodes, rest, nil
}
