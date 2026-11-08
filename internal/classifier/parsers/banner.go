package parsers

import (
	"unicode"

	"github.com/bitmagnet-io/bitmagnet/internal/keywords"
	"github.com/bitmagnet-io/bitmagnet/internal/regex"
	"github.com/hedhyw/rex/pkg/dialect"
	"github.com/hedhyw/rex/pkg/rex"
)

var domainLabelToken = rex.Common.Class(
	rex.Chars.Alphanumeric(),
	rex.Chars.Single('-'),
).Repeat().OneOrMore()

var domainTLDToken = rex.Common.Class(rex.Chars.Alphabetic()).Repeat().Between(2, 6)

// domainToken matches a domain name with any TLD, e.g. "www.UIndex.org" or "Ex-torrenty.org".
var domainToken = rex.Group.Define(
	domainLabelToken,
	rex.Group.NonCaptured(rex.Chars.Single('.'), domainLabelToken).Repeat().ZeroOrMore(),
	rex.Chars.Single('.'),
	domainTLDToken,
).NonCaptured()

// siteBannerRegex matches a leading site banner, e.g. "www.UIndex.org - ", "Torrenting.com - " or "[Ex-torrenty.org]".
// Without a "www." prefix or brackets, the TLD is restricted to a known list,
// to avoid stripping dotted titles such as "Dr.Who - ".
var siteBannerRegex = rex.New(
	rex.Chars.Begin(),
	rex.Chars.Whitespace().Repeat().ZeroOrMore(),
	rex.Group.Composite(
		rex.Group.Define(
			rex.Group.Composite(
				rex.Group.Define(
					rex.Group.Composite(keywords.MustNewRexTokensFromKeywords("www")...).NonCaptured(),
					rex.Chars.Single('.'),
					domainToken,
				).NonCaptured(),
				rex.Group.Define(
					domainLabelToken,
					rex.Chars.Single('.'),
					rex.Group.Composite(keywords.MustNewRexTokensFromKeywords(
						"com", "org", "net", "info", "tv", "cc", "io", "xyz", "ws", "se", "ru", "pl", "nz", "li", "cx", "la", "sx",
					)...).NonCaptured(),
				).NonCaptured(),
			).NonCaptured(),
			rex.Chars.Whitespace().Repeat().ZeroOrMore(),
			rex.Chars.Single('-'),
		).NonCaptured(),
		rex.Group.Define(
			rex.Chars.Single('['),
			domainToken,
			rex.Chars.Single(']'),
		).NonCaptured(),
	).NonCaptured(),
	rex.Chars.Whitespace().Repeat().ZeroOrMore(),
).MustCompile()

func cjkCharToken() dialect.Token {
	return rex.Common.Class(
		rex.Chars.Unicode(unicode.Han),
		rex.Chars.Unicode(unicode.Hiragana),
		rex.Chars.Unicode(unicode.Katakana),
		rex.Chars.Unicode(unicode.Hangul),
	)
}

var cjkBracketsToken = rex.Group.Define(
	rex.Chars.Single('【'),
	rex.Common.NotClass(rex.Chars.Single('】')).Repeat().ZeroOrMore(),
	rex.Chars.Single('】'),
).NonCaptured()

// cjkPrefixRegex matches leading CJK text (including bracketed banners such as "【… www.HDBTHD.com】")
// that is followed by a Latin title, e.g. "防弹武僧[中文字幕].Bulletproof.Monk.2003"; the Latin letter is captured.
var cjkPrefixRegex = rex.New(
	rex.Chars.Begin(),
	rex.Group.Composite(cjkBracketsToken, cjkCharToken()).NonCaptured(),
	rex.Group.Composite(
		cjkBracketsToken,
		rex.Group.Define(
			rex.Chars.Single('['),
			rex.Common.NotClass(rex.Chars.Single(']')).Repeat().ZeroOrMore(),
			rex.Chars.Single(']'),
		).NonCaptured(),
		cjkCharToken(),
		regex.AnyNonWordChar(),
	).NonCaptured().Repeat().ZeroOrMore(),
	rex.Group.Define(rex.Chars.Unicode(unicode.Latin)),
).MustCompile()

// stripReleaseNameBanner removes leading site banners and CJK titles preceding a Latin title from a release name.
func stripReleaseNameBanner(name string) string {
	name = siteBannerRegex.ReplaceAllString(name, "")
	name = cjkPrefixRegex.ReplaceAllString(name, "$1")

	return name
}
