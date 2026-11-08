package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinNormalizeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		a, b string
	}{
		{"Frieren: Beyond Journey's End", "Frieren Beyond Journeys End"},
		{"Frieren: Beyond Journey’s End", "Frieren Beyond Journeys End"},
		{"Mairimashita! Iruma-kun", "Mairimashita Iruma kun"},
		{"Стражи Галактики", "Strazhi galaktiki"},
	}

	for _, tc := range testCases {
		t.Run(tc.a, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, levenshteinNormalizeString(tc.a), levenshteinNormalizeString(tc.b))
		})
	}
}
