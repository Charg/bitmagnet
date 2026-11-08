package adapter

import (
	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/database/search"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
)

// Anime is identified as TV content originally in Japanese with the TMDB Animation genre.
var (
	animeLanguage   = model.Language("ja")
	animeCollection = model.ContentCollectionRef{
		Type:   "genre",
		Source: "tmdb",
		ID:     "16",
	}
)

func animeCriteria() query.Criteria {
	return query.And(
		search.ContentOriginalLanguageCriteria(animeLanguage),
		search.ContentCollectionCriteria(animeCollection),
	)
}

func isAnime(content model.Content) bool {
	if !content.OriginalLanguage.Valid || content.OriginalLanguage.Language != animeLanguage {
		return false
	}

	for _, c := range content.Collections {
		if c.Type == animeCollection.Type && c.Source == animeCollection.Source && c.ID == animeCollection.ID {
			return true
		}
	}

	return false
}
