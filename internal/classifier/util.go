package classifier

import (
	"strings"
	"unicode"

	"github.com/agnivade/levenshtein"
	"github.com/bitmagnet-io/bitmagnet/internal/regex"
	"github.com/mozillazg/go-unidecode"
)

const levenshteinThreshold = 5

func levenshteinFindBestMatch[T any](target string, items []T, getCandidates func(T) []string) (t T, ok bool) {
	minDistance := levenshteinThreshold + 1
	bestMatch := -1

	for i, item := range items {
		candidates := getCandidates(item)

		distance := levenshteinFindMinDistance(target, candidates)
		if distance >= 0 && distance < minDistance {
			minDistance = distance
			bestMatch = i

			if distance == 0 {
				break
			}
		}
	}

	if bestMatch == -1 {
		return t, false
	}

	return items[bestMatch], true
}

func levenshteinIsMatch(target string, candidates []string) bool {
	distance := levenshteinFindMinDistance(target, candidates)
	return distance >= 0 && distance <= levenshteinThreshold
}

func levenshteinFindMinDistance(target string, candidates []string) int {
	normTarget := levenshteinNormalizeString(target)
	triedCandidates := make(map[string]struct{}, len(candidates))
	minDistance := -1

	for _, candidate := range candidates {
		normCandidate := levenshteinNormalizeString(candidate)
		if _, ok := triedCandidates[normCandidate]; ok {
			continue
		}

		distance := levenshtein.ComputeDistance(normTarget, normCandidate)
		if minDistance == -1 || distance < minDistance {
			minDistance = distance
		}

		triedCandidates[normCandidate] = struct{}{}
	}

	return minDistance
}

// levenshteinNormalizeString transliterates and lowercases the string, removes apostrophes and replaces other
// punctuation with spaces, so that e.g. "Journey's" matches "Journeys" and "Iruma-kun" matches "Iruma kun".
func levenshteinNormalizeString(str string) string {
	str = strings.Map(func(r rune) rune {
		switch {
		case r == '\'':
			return -1
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return r
		default:
			return ' '
		}
	}, regex.NormalizeString(unidecode.Unidecode(str)))

	return strings.Join(strings.Fields(str), " ")
}
