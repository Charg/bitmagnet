package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEpisodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  Episodes
	}{
		{"S01", Episodes{1: {}}},
		{"S01E12", Episodes{1: {12: {}}}},
		{"S01E01-03", Episodes{1: {1: {}, 2: {}, 3: {}}}},
		{"S01E01-E03", Episodes{1: {1: {}, 2: {}, 3: {}}}},
		{"S01E01,E03", Episodes{1: {1: {}, 3: {}}}},
		{"S01, S03", Episodes{1: {}, 3: {}}},
		{"1x05", Episodes{1: {5: {}}}},
		{"1x05-07", Episodes{1: {5: {}, 6: {}, 7: {}}}},
		{"S01-S03", Episodes{1: {}, 2: {}, 3: {}}},
		{"S01-03", Episodes{1: {}, 2: {}, 3: {}}},
		{"S01 - S03", Episodes{1: {}, 2: {}, 3: {}}},
		{"Season 1-3", Episodes{1: {}, 2: {}, 3: {}}},
		{"Season.3", Episodes{3: {}}},
		{"Season_3", Episodes{3: {}}},
		{"S2 - 05", Episodes{2: {5: {}}}},
		{"S151E04", Episodes{151: {4: {}}}},
		{"S100", Episodes{100: {}}},
		{"S2 - 05v2", Episodes{2: {5: {}}}},
		{"S2 - 105", Episodes{2: {105: {}}}},
		{"Season 2 - 03", Episodes{2: {3: {}}}},
		{"S07E06v2", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ParseEpisodes(tt.input))
		})
	}
}
