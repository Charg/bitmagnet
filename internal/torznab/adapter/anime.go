package adapter

import (
	"slices"

	"github.com/bitmagnet-io/bitmagnet/internal/database/query"
	"github.com/bitmagnet-io/bitmagnet/internal/database/search"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
)

// Anime is identified as TV content with the TMDB Animation genre, originally in Japanese, Chinese or Korean
// (*arr apps treat Chinese donghua and Korean animation as anime too).
var (
	animeLanguages  = []model.Language{"ja", "zh", "ko"}
	animeCollection = model.ContentCollectionRef{
		Type:   "genre",
		Source: "tmdb",
		ID:     "16",
	}
)

func animeCriteria() query.Criteria {
	return query.And(
		search.ContentOriginalLanguageCriteria(animeLanguages...),
		search.ContentCollectionCriteria(animeCollection),
	)
}

func isAnime(content model.Content) bool {
	if !content.OriginalLanguage.Valid || !slices.Contains(animeLanguages, content.OriginalLanguage.Language) {
		return false
	}

	for _, c := range content.Collections {
		if c.Type == animeCollection.Type && c.Source == animeCollection.Source && c.ID == animeCollection.ID {
			return true
		}
	}

	return false
}
