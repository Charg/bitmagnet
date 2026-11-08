package adapter

import (
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/database/search"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/torznab"
	"github.com/stretchr/testify/assert"
)

func attrValues(item torznab.SearchResultItem, name string) []string {
	var values []string

	for _, attr := range item.TorznabAttrs {
		if attr.AttrName == name {
			values = append(values, attr.AttrValue)
		}
	}

	return values
}

func contentWithIDs(ct model.ContentType, tmdbID string, ids map[string]string) model.Content {
	content := model.Content{
		Type:   ct,
		Source: "tmdb",
		ID:     tmdbID,
	}

	for source, id := range ids {
		content.Attributes = append(content.Attributes, model.ContentAttribute{
			Source: source,
			Key:    "id",
			Value:  id,
		})
	}

	return content
}

func resultItem(ct model.ContentType, content model.Content) search.TorrentContentResultItem {
	return search.TorrentContentResultItem{
		TorrentContent: model.TorrentContent{
			ContentType: model.NewNullContentType(ct),
			Content:     content,
		},
	}
}

func TestTorrentContentResultItemIMDBAttr(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		imdbID   string
		expected []string
	}{
		{name: "prefixed", imdbID: "tt6751668", expected: []string{"6751668"}},
		{name: "unprefixed", imdbID: "6751668", expected: []string{"6751668"}},
		{name: "short", imdbID: "t", expected: []string{"t"}},
		{name: "prefix only", imdbID: "tt", expected: nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			item := torrentContentResultItemToTorznabResultItem(resultItem(
				model.ContentTypeMovie,
				contentWithIDs(model.ContentTypeMovie, "496243", map[string]string{"imdb": testCase.imdbID}),
			))

			assert.Equal(t, testCase.expected, attrValues(item, torznab.AttrImdb))
		})
	}
}

func TestTorrentContentResultItemIDAttrs(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		item     search.TorrentContentResultItem
		expected map[string][]string
	}{
		{
			name: "tv show",
			item: resultItem(model.ContentTypeTvShow, contentWithIDs(
				model.ContentTypeTvShow, "250923", map[string]string{"imdb": "tt26933824", "tvdb": "421069"},
			)),
			expected: map[string][]string{
				torznab.AttrImdb:   {"26933824"},
				torznab.AttrTmdb:   {"250923"},
				torznab.AttrTmdbID: {"250923"},
				torznab.AttrTvdbID: {"421069"},
			},
		},
		{
			name: "movie",
			item: resultItem(model.ContentTypeMovie, contentWithIDs(
				model.ContentTypeMovie, "496243", map[string]string{"imdb": "tt6751668", "tvdb": "123"},
			)),
			expected: map[string][]string{
				torznab.AttrImdb:   {"6751668"},
				torznab.AttrTmdb:   {"496243"},
				torznab.AttrTmdbID: {"496243"},
				torznab.AttrTvdbID: nil,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			item := torrentContentResultItemToTorznabResultItem(testCase.item)

			for name, values := range testCase.expected {
				assert.Equal(t, values, attrValues(item, name), name)
			}
		})
	}
}
