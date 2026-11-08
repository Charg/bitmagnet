package parsers

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/bitmagnet-io/bitmagnet/internal/classifier/classification"
	"github.com/bitmagnet-io/bitmagnet/internal/keywords"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/regex"
	"github.com/hedhyw/rex/pkg/dialect"
	"github.com/hedhyw/rex/pkg/rex"
)

var titleTokens = []dialect.Token{
	rex.Group.Define(
		rex.Group.Composite(
			rex.Group.NonCaptured(
				regex.AnyWordChar().Repeat().OneOrMore(),
				rex.Group.NonCaptured(
					rex.Chars.Single('-'), regex.AnyWordChar().Repeat().OneOrMore(),
				).Repeat().ZeroOrMore(),
			),
			regex.AnyNonWordChar().Repeat().OneOrMore(),
		).NonCaptured().Repeat().OneOrMore(),
		rex.Group.Composite(
			regex.AnyNonWordChar().Repeat().OneOrMore(),
			rex.Chars.End(),
		).NonCaptured(),
	),
}

var titleRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(titleTokens...),
).MustCompile()

var yearTokens = []dialect.Token{
	rex.Group.NonCaptured(rex.Common.NotClass(rex.Chars.WordCharacter()).Repeat().ZeroOrMore()),
	rex.Group.Define(
		rex.Group.Composite(
			rex.Common.Text("18"), rex.Common.Text("19"), rex.Common.Text("20"),
		).NonCaptured(),
		rex.Chars.Digits().Repeat().Exactly(2),
	),
	rex.Group.Composite(
		rex.Common.NotClass(rex.Chars.WordCharacter()),
		rex.Chars.End(),
	).NonCaptured(),
}

var titleYearRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(rex.Group.NonCaptured(titleTokens...), rex.Group.NonCaptured(yearTokens...)),
).MustCompile()

var titleEpisodesRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.NonCaptured(
		rex.Group.NonCaptured(titleTokens...),
		model.EpisodesToken,
	),
).MustCompile()

var leadingEpisodesRegex = rex.New(
	rex.Chars.Begin(),
	model.EpisodesToken,
).MustCompile()

var multiRegex = keywords.MustNewRegexFromKeywords("multi", "dual")

var separatorToken = rex.Chars.Runes(" ._")

var titlePartRegex = rex.New(
	separatorToken.Repeat().ZeroOrOne(),
	rex.Group.Define(regex.WordToken()),
	separatorToken.Repeat().ZeroOrOne(),
).MustCompile()

var trimTitleRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.Composite(
		rex.Group.NonCaptured(
			rex.Chars.Single('['),
			rex.Common.NotClass(rex.Chars.Single(']')).Repeat().OneOrMore(),
			rex.Chars.Single(']'),
		),
		rex.Group.NonCaptured(
			rex.Chars.Single('【'),
			rex.Common.NotClass(rex.Chars.Single('】')).Repeat().OneOrMore(),
			rex.Chars.Single('】'),
		),
	).NonCaptured().Repeat().ZeroOrOne(),
	regex.AnyNonWordChar().Repeat().ZeroOrMore(),
	rex.Group.Define(
		regex.WordToken(),
		rex.Group.NonCaptured(
			rex.Chars.Any(),
			regex.WordToken(),
		).Repeat().ZeroOrMore(),
	),
	regex.AnyNonWordChar().Repeat().ZeroOrMore(),
	rex.Chars.End(),
).MustCompile()

func cleanTitle(title string) string {
	title = titlePartRegex.ReplaceAllStringFunc(title, func(s string) string {
		partMatch := titlePartRegex.FindStringSubmatch(s)
		if partMatch == nil {
			return ""
		}

		return partMatch[1] + " "
	})
	title = trimTitleRegex.ReplaceAllString(title, "$1")

	return title
}

func parseTitleYear(input string) (string, model.Year, string, error) {
	if match := titleYearRegex.FindStringSubmatch(input); match != nil {
		yearMatch, _ := strconv.ParseUint(match[2], 10, 16)
		title := cleanTitle(match[1])

		if title != "" {
			return title, model.Year(yearMatch), input[len(match[0]):], nil
		}
	}

	return "", 0, "", classification.ErrUnmatched
}

func parseTitle(input string) (title string, rest string, err error) {
	if match := titleRegex.FindStringSubmatch(input); match != nil {
		title = cleanTitle(match[1])
		if title != "" {
			return title, input[len(match[0]):], nil
		}
	}

	return "", "", classification.ErrUnmatched
}

// widenEpisodesMatch handles the greedy title match consuming the start of a range,
// e.g. "Show.S01-S05" matching the title "Show.S01-" and the episodes "S05":
// it returns the earliest episodes match within the title that runs into the original episodes match.
func widenEpisodesMatch(input string, title string, match []string) (string, []string) {
	for i := 1; i < len(title); i++ {
		if (title[i] != 's' && title[i] != 'S') || isWordByte(title[i-1]) {
			continue
		}

		idx := leadingEpisodesRegex.FindStringSubmatchIndex(input[i:])
		if idx != nil && i+idx[1] > len(title) && !seasonNumberTruncated(input[i:], idx[6:10]) {
			m := submatches(input[i:], idx)

			return input[:i], append([]string{input[:i+idx[1]], input[:i]}, m[1:]...)
		}
	}

	return title, match
}

// seasonNumberTruncated reports whether a matched season number (given as start/end index pairs of the
// season start and end groups) is immediately followed by another digit, e.g. "Season.2022-23" matching season 20.
func seasonNumberTruncated(input string, seasonIdx []int) bool {
	for i := 1; i < len(seasonIdx); i += 2 {
		if end := seasonIdx[i]; end >= 0 && end < len(input) && unicode.IsDigit(rune(input[end])) {
			return true
		}
	}

	return false
}

func submatches(input string, idx []int) []string {
	match := make([]string, len(idx)/2)

	for i := range match {
		if idx[2*i] >= 0 {
			match[i] = input[idx[2*i]:idx[2*i+1]]
		}
	}

	return match
}

func isWordByte(b byte) bool {
	return b >= utf8.RuneSelf || unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
}

func parseTitleYearEpisodes(input string) (string, model.Year, model.Episodes, string, error) {
	if idx := titleEpisodesRegex.FindStringSubmatchIndex(input); idx != nil && !seasonNumberTruncated(input, idx[8:12]) {
		match := submatches(input, idx)
		title, match := widenEpisodesMatch(input, match[1], match)
		year := model.Year(0)

		if t, y, _, err := parseTitleYear(title); err == nil {
			title = t
			year = y
		} else {
			title = cleanTitle(title)
		}

		episodes := model.EpisodesMatchToEpisodes(match[2:])

		return title, year, episodes, input[len(match[0]):], nil
	}

	return "", 0, nil, "", classification.ErrUnmatched
}

func ParseTitleYearEpisodes(
	contentType model.NullContentType,
	input string,
) (string, model.Year, model.Episodes, string, error) {
	if !contentType.Valid || contentType.ContentType == model.ContentTypeTvShow {
		if title, year, episodes, rest, err := parseTitleYearEpisodes(input); err == nil {
			return title, year, episodes, rest, nil
		}
	}

	if title, year, rest, err := parseTitleYear(input); err == nil {
		return title, year, nil, rest, nil
	}

	if title, rest, err := parseTitle(input); err == nil {
		return title, 0, nil, rest, nil
	}

	return "", 0, nil, "", classification.ErrUnmatched
}

func ParseVideoContent(torrent model.Torrent, result classification.Result) (classification.ContentAttributes, error) {
	name := stripReleaseNameBanner(torrent.Name)

	var (
		title    string
		year     model.Year
		episodes model.Episodes
		rest     string
		err      error
		absolute bool
	)

	if !result.ContentType.Valid || result.ContentType.ContentType == model.ContentTypeTvShow {
		title, year, episodes, rest, err = parseAbsoluteEpisode(name)
		absolute = err == nil
	}

	if !absolute {
		title, year, episodes, rest, err = ParseTitleYearEpisodes(result.ContentType, name)
	}

	if err != nil {
		if !result.ContentType.Valid {
			return classification.ContentAttributes{}, err
		}

		rest = torrent.Name
	}

	ct := model.NullContentType{}

	switch {
	case result.ContentType.Valid:
		ct = model.NullContentType{Valid: true, ContentType: result.ContentType.ContentType}
	case len(episodes) > 0 || absolute || result.Date.IsValid():
		ct = model.NullContentType{Valid: true, ContentType: model.ContentTypeTvShow}
	case !year.IsNil():
		ct = model.NullContentType{Valid: true, ContentType: model.ContentTypeMovie}
	}

	if ct.ContentType != model.ContentTypeTvShow {
		episodes = nil

		if year.IsNil() {
			title = ""
			rest = torrent.Name
		}
	}

	attrs := classification.ContentAttributes{
		ContentType:   ct,
		BaseTitle:     model.NullString{Valid: title != "", String: title},
		Date:          model.Date{Year: year},
		Episodes:      episodes,
		Languages:     model.InferLanguages(rest),
		LanguageMulti: multiRegex.MatchString(rest),
	}
	attrs.InferVideoAttributes(rest)

	return attrs, nil
}
