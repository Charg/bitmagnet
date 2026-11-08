package classifier

import (
	"errors"

	"github.com/bitmagnet-io/bitmagnet/internal/classifier/classification"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/tmdb"
)

// tmdbAlternativeTitlesMaxResults bounds the number of extra details requests made when no search result
// matches on its primary titles. TMDB search matches alternative titles server-side (e.g. romanised anime titles),
// so the correct result is usually amongst the first few even when its primary titles don't match the query.
const tmdbAlternativeTitlesMaxResults = 3

func (c executionContext) tmdbSearchMovie(title string, year model.Year) (model.Content, error) {
	req := tmdb.SearchMovieRequest{
		Query:        title,
		IncludeAdult: true,
	}
	if !year.IsNil() {
		req.Year = year
	}

	searchResult, searchErr := c.tmdbClient.SearchMovie(c.Context, req)
	if searchErr != nil {
		return model.Content{}, searchErr
	}

	bestMatch, ok := levenshteinFindBestMatch[tmdb.SearchMovieResult](
		title,
		searchResult.Results,
		func(item tmdb.SearchMovieResult) []string {
			return []string{item.Title, item.OriginalTitle}
		},
	)

	if !ok {
		return c.tmdbMatchMovieByAlternativeTitles(title, searchResult.Results)
	}

	return c.tmdbGetMovieByTMDBID(bestMatch.ID)
}

// tmdbMatchMovieByAlternativeTitles fetches details including alternative titles for the top search results,
// returning the first result with a title matching the query.
func (c executionContext) tmdbMatchMovieByAlternativeTitles(
	title string,
	results []tmdb.SearchMovieResult,
) (model.Content, error) {
	for _, result := range results[:min(len(results), tmdbAlternativeTitlesMaxResults)] {
		d, err := c.tmdbClient.MovieDetails(c.Context, tmdb.MovieDetailsRequest{
			ID:               result.ID,
			AppendToResponse: []string{"alternative_titles"},
		})
		if err != nil {
			if errors.Is(err, tmdb.ErrNotFound) {
				continue
			}

			return model.Content{}, err
		}

		candidates := []string{d.Title, d.OriginalTitle}
		for _, t := range d.AlternativeTitles.Titles {
			candidates = append(candidates, t.Title)
		}

		if levenshteinIsMatch(title, candidates) {
			return tmdb.MovieDetailsToMovieModel(d)
		}
	}

	return model.Content{}, classification.ErrUnmatched
}

func (c executionContext) tmdbSearchTVShow(title string, year model.Year) (model.Content, error) {
	req := tmdb.SearchTvRequest{
		Query:        title,
		IncludeAdult: true,
	}
	if !year.IsNil() {
		req.FirstAirDateYear = year
	}

	searchResult, searchErr := c.tmdbClient.SearchTv(c.Context, req)
	if searchErr != nil {
		return model.Content{}, searchErr
	}

	bestMatch, ok := levenshteinFindBestMatch[tmdb.SearchTvResult](
		title,
		searchResult.Results,
		func(item tmdb.SearchTvResult) []string {
			return []string{item.Name, item.OriginalName}
		},
	)

	if !ok {
		return c.tmdbMatchTVShowByAlternativeTitles(title, searchResult.Results)
	}

	return c.tmdbGetTVShowByTMDBID(bestMatch.ID)
}

// tmdbMatchTVShowByAlternativeTitles fetches details including alternative titles for the top search results,
// returning the first result with a title matching the query.
func (c executionContext) tmdbMatchTVShowByAlternativeTitles(
	title string,
	results []tmdb.SearchTvResult,
) (model.Content, error) {
	for _, result := range results[:min(len(results), tmdbAlternativeTitlesMaxResults)] {
		d, err := c.tmdbClient.TvDetails(c.Context, tmdb.TvDetailsRequest{
			SeriesID:         result.ID,
			AppendToResponse: []string{"external_ids", "alternative_titles"},
		})
		if err != nil {
			if errors.Is(err, tmdb.ErrNotFound) {
				continue
			}

			return model.Content{}, err
		}

		candidates := []string{d.Name, d.OriginalName}
		for _, t := range d.AlternativeTitles.Results {
			candidates = append(candidates, t.Title)
		}

		if levenshteinIsMatch(title, candidates) {
			return tmdb.TvShowDetailsToTvShowModel(d)
		}
	}

	return model.Content{}, classification.ErrUnmatched
}

func (c executionContext) tmdbGetMovieByTMDBID(id int64) (movie model.Content, err error) {
	d, getDetailsErr := c.tmdbClient.MovieDetails(c.Context, tmdb.MovieDetailsRequest{
		ID: id,
	})
	if getDetailsErr != nil {
		if errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			getDetailsErr = classification.ErrUnmatched
		}

		err = getDetailsErr

		return
	}

	return tmdb.MovieDetailsToMovieModel(d)
}

func (c executionContext) tmdbGetTVShowByTMDBID(id int64) (movie model.Content, err error) {
	d, getDetailsErr := c.tmdbClient.TvDetails(c.Context, tmdb.TvDetailsRequest{
		SeriesID:         id,
		AppendToResponse: []string{"external_ids"},
	})
	if getDetailsErr != nil {
		if errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			getDetailsErr = classification.ErrUnmatched
		}

		err = getDetailsErr

		return
	}

	return tmdb.TvShowDetailsToTvShowModel(d)
}

func (c executionContext) tmdbGetTMDBIDByExternalID(ref model.ContentRef) (int64, error) {
	externalSource, externalID, externalSourceErr := tmdb.ExternalSource(ref)
	if externalSourceErr != nil {
		return 0, externalSourceErr
	}

	byIDResult, byIDErr := c.tmdbClient.FindByID(c.Context, tmdb.FindByIDRequest{
		ExternalSource: externalSource,
		ExternalID:     externalID,
	})
	if byIDErr != nil {
		return 0, byIDErr
	}

	switch ref.Type {
	case model.ContentTypeMovie, model.ContentTypeXxx:
		if len(byIDResult.MovieResults) == 0 {
			return 0, classification.ErrUnmatched
		}

		return byIDResult.MovieResults[0].ID, nil
	case model.ContentTypeTvShow:
		if len(byIDResult.TvResults) == 0 {
			return 0, classification.ErrUnmatched
		}

		return byIDResult.TvResults[0].ID, nil
	default:
		return 0, classification.ErrUnmatched
	}
}
