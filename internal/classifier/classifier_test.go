package classifier

import (
	"context"
	"fmt"
	"testing"

	"github.com/bitmagnet-io/bitmagnet/internal/classifier/classification"
	classifier_mocks "github.com/bitmagnet-io/bitmagnet/internal/classifier/mocks"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/tmdb"
	tmdb_mocks "github.com/bitmagnet-io/bitmagnet/internal/tmdb/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClassifier(t *testing.T) {
	t.Parallel()

	matchContext := mock.MatchedBy(func(ctx any) bool {
		_, ok := ctx.(context.Context)
		return ok
	})

	testCases := []struct {
		torrent      model.Torrent
		flags        Flags
		prepareMocks func(mocks testClassifierMocks)
		expected     classification.Result
		expectedErr  error
	}{
		{
			torrent: model.Torrent{
				Name:        "The Regular Movie (2000).mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeMovie,
					"The Regular Movie",
					model.Year(2000),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchMovie",
					matchContext,
					tmdb.SearchMovieRequest{
						Query:        "The Regular Movie",
						Year:         2000,
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchMovieResponse{}, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType: model.NewNullContentType(model.ContentTypeMovie),
					BaseTitle:   model.NewNullString("The Regular Movie"),
					Date: model.Date{
						Year: 2000,
					},
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "The Regular Local Movie (2000).mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeMovie,
					"The Regular Local Movie",
					model.Year(2000),
				).
					Return(model.Content{
						Type:        model.ContentTypeMovie,
						Source:      "local",
						ID:          "123",
						Title:       "The Regular Local Movie",
						ReleaseYear: 2000,
					}, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType: model.NewNullContentType(model.ContentTypeMovie),
					BaseTitle:   model.NewNullString("The Regular Local Movie"),
					Date: model.Date{
						Year: 2000,
					},
				},
				Content: &model.Content{
					Type:        model.ContentTypeMovie,
					Source:      "local",
					ID:          "123",
					Title:       "The Regular Local Movie",
					ReleaseYear: 2000,
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "The Regular TMDB Movie (2000).mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeMovie,
					"The Regular TMDB Movie",
					model.Year(2000),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchMovie",
					matchContext,
					tmdb.SearchMovieRequest{
						Query:        "The Regular TMDB Movie",
						Year:         2000,
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchMovieResponse{
						Results: []tmdb.SearchMovieResult{
							{
								ID:          123,
								Title:       "The Regular TMDB Movie",
								ReleaseDate: "2000-01-01",
							},
						},
					}, nil)
				mocks.tmdbClient.On(
					"MovieDetails",
					matchContext,
					tmdb.MovieDetailsRequest{
						ID: 123,
					},
				).
					Return(tmdb.MovieDetailsResponse{
						ID:            123,
						Title:         "The Regular TMDB Movie",
						OriginalTitle: "The Regular TMDB Movie Original",
						ReleaseDate:   "2000-01-01",
					}, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType: model.NewNullContentType(model.ContentTypeMovie),
					BaseTitle:   model.NewNullString("The Regular TMDB Movie"),
					Date: model.Date{
						Year: 2000,
					},
				},
				Content: &model.Content{
					Type:   model.ContentTypeMovie,
					Source: "tmdb",
					ID:     "123",
					Title:  "The Regular TMDB Movie",
					ReleaseDate: model.Date{
						Year:  2000,
						Month: 1,
						Day:   1,
					},
					ReleaseYear:   2000,
					Adult:         model.NewNullBool(false),
					OriginalTitle: model.NewNullString("The Regular TMDB Movie Original"),
					Popularity:    model.NewNullFloat32(0),
					VoteAverage:   model.NewNullFloat32(0),
					VoteCount:     model.NewNullUint(0),
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "Frieren Beyond Journeys End S01E10 1080p.mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeTvShow,
					"Frieren Beyond Journeys End",
					model.Year(0),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchTv",
					matchContext,
					tmdb.SearchTvRequest{
						Query:        "Frieren Beyond Journeys End",
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchTvResponse{
						Results: []tmdb.SearchTvResult{frierenSearchResult},
					}, nil)
				// the primary match succeeds, so alternative titles must not be requested:
				mocks.tmdbClient.On(
					"TvDetails",
					matchContext,
					tmdb.TvDetailsRequest{
						SeriesID:         209867,
						AppendToResponse: []string{"external_ids"},
					},
				).
					Return(frierenDetails, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType:     model.NewNullContentType(model.ContentTypeTvShow),
					BaseTitle:       model.NewNullString("Frieren Beyond Journeys End"),
					Episodes:        model.Episodes{1: {10: {}}},
					VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
				},
				Content: &frierenContent,
			},
		},
		{
			torrent: model.Torrent{
				Name:        "Sousou no Frieren S01E05 1080p.mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeTvShow,
					"Sousou no Frieren",
					model.Year(0),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchTv",
					matchContext,
					tmdb.SearchTvRequest{
						Query:        "Sousou no Frieren",
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchTvResponse{
						Results: []tmdb.SearchTvResult{frierenSearchResult},
					}, nil)
				mocks.tmdbClient.On(
					"TvDetails",
					matchContext,
					tmdb.TvDetailsRequest{
						SeriesID:         209867,
						AppendToResponse: []string{"external_ids", "alternative_titles"},
					},
				).
					Return(frierenDetailsWithAlternativeTitles, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType:     model.NewNullContentType(model.ContentTypeTvShow),
					BaseTitle:       model.NewNullString("Sousou no Frieren"),
					Episodes:        model.Episodes{1: {5: {}}},
					VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
				},
				Content: &frierenContent,
			},
		},
		{
			torrent: model.Torrent{
				Name:        "Boku no Hero Academia S07E01 1080p.mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeTvShow,
					"Boku no Hero Academia",
					model.Year(0),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchTv",
					matchContext,
					tmdb.SearchTvRequest{
						Query:        "Boku no Hero Academia",
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchTvResponse{
						Results: []tmdb.SearchTvResult{
							{ID: 1, Name: "Vigilantes", OriginalName: "ヴィジランテ"},
							{ID: 65930, Name: "My Hero Academia", OriginalName: "僕のヒーローアカデミア"},
						},
					}, nil)
				mocks.tmdbClient.On(
					"TvDetails",
					matchContext,
					tmdb.TvDetailsRequest{
						SeriesID:         1,
						AppendToResponse: []string{"external_ids", "alternative_titles"},
					},
				).
					Return(tmdb.TvDetailsResponse{
						ID:           1,
						Name:         "Vigilantes",
						OriginalName: "ヴィジランテ",
						AlternativeTitles: tmdb.TvAlternativeTitles{
							Results: []tmdb.AlternativeTitle{{Iso3166_1: "JP", Title: "Vigilante: Boku no Hero Academia Illegals"}},
						},
					}, nil)
				mocks.tmdbClient.On(
					"TvDetails",
					matchContext,
					tmdb.TvDetailsRequest{
						SeriesID:         65930,
						AppendToResponse: []string{"external_ids", "alternative_titles"},
					},
				).
					Return(tmdb.TvDetailsResponse{
						ID:           65930,
						Name:         "My Hero Academia",
						OriginalName: "僕のヒーローアカデミア",
						AlternativeTitles: tmdb.TvAlternativeTitles{
							Results: []tmdb.AlternativeTitle{{Iso3166_1: "JP", Title: "Boku no Hero Academia"}},
						},
					}, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType:     model.NewNullContentType(model.ContentTypeTvShow),
					BaseTitle:       model.NewNullString("Boku no Hero Academia"),
					Episodes:        model.Episodes{7: {1: {}}},
					VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
				},
				Content: &model.Content{
					Type:          model.ContentTypeTvShow,
					Source:        "tmdb",
					ID:            "65930",
					Title:         "My Hero Academia",
					OriginalTitle: model.NewNullString("僕のヒーローアカデミア"),
					Popularity:    model.NewNullFloat32(0),
					VoteAverage:   model.NewNullFloat32(0),
					VoteCount:     model.NewNullUint(0),
					Collections:   []model.ContentCollection{},
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "Shin no Nakama S01E01 1080p.mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeTvShow,
					"Shin no Nakama",
					model.Year(0),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchTv",
					matchContext,
					tmdb.SearchTvRequest{
						Query:        "Shin no Nakama",
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchTvResponse{
						Results: []tmdb.SearchTvResult{
							{ID: 1, Name: "Unrelated One"},
							{ID: 2, Name: "Unrelated Two"},
							{ID: 3, Name: "Unrelated Three"},
							{ID: 4, Name: "Unrelated Four"},
						},
					}, nil)
				// only the top 3 results are checked for alternative titles, and a missing result is skipped:
				for _, id := range []int64{1, 3} {
					mocks.tmdbClient.On(
						"TvDetails",
						matchContext,
						tmdb.TvDetailsRequest{
							SeriesID:         id,
							AppendToResponse: []string{"external_ids", "alternative_titles"},
						},
					).
						Return(tmdb.TvDetailsResponse{
							ID:   id,
							Name: "Unrelated",
							AlternativeTitles: tmdb.TvAlternativeTitles{
								Results: []tmdb.AlternativeTitle{{Title: "Something Else Entirely"}},
							},
						}, nil)
				}
				mocks.tmdbClient.On(
					"TvDetails",
					matchContext,
					tmdb.TvDetailsRequest{
						SeriesID:         2,
						AppendToResponse: []string{"external_ids", "alternative_titles"},
					},
				).
					Return(tmdb.TvDetailsResponse{}, tmdb.ErrNotFound)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType:     model.NewNullContentType(model.ContentTypeTvShow),
					BaseTitle:       model.NewNullString("Shin no Nakama"),
					Episodes:        model.Episodes{1: {1: {}}},
					VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "Strazhi galaktiki (2014).mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			prepareMocks: func(mocks testClassifierMocks) {
				mocks.search.On(
					"ContentBySearch",
					matchContext,
					model.ContentTypeMovie,
					"Strazhi galaktiki",
					model.Year(2014),
				).
					Return(model.Content{}, classification.ErrUnmatched)
				mocks.tmdbClient.On(
					"SearchMovie",
					matchContext,
					tmdb.SearchMovieRequest{
						Query:        "Strazhi galaktiki",
						Year:         2014,
						IncludeAdult: true,
					},
				).
					Return(tmdb.SearchMovieResponse{
						Results: []tmdb.SearchMovieResult{
							{
								ID:            118340,
								Title:         "Guardians of the Galaxy",
								OriginalTitle: "Guardians of the Galaxy",
								ReleaseDate:   "2014-07-30",
							},
						},
					}, nil)
				mocks.tmdbClient.On(
					"MovieDetails",
					matchContext,
					tmdb.MovieDetailsRequest{
						ID:               118340,
						AppendToResponse: []string{"alternative_titles"},
					},
				).
					Return(tmdb.MovieDetailsResponse{
						ID:            118340,
						Title:         "Guardians of the Galaxy",
						OriginalTitle: "Guardians of the Galaxy",
						ReleaseDate:   "2014-07-30",
						AlternativeTitles: tmdb.MovieAlternativeTitles{
							// matched after transliteration of the Cyrillic title:
							Titles: []tmdb.AlternativeTitle{{Iso3166_1: "RU", Title: "Стражи Галактики"}},
						},
					}, nil)
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType: model.NewNullContentType(model.ContentTypeMovie),
					BaseTitle:   model.NewNullString("Strazhi galaktiki"),
					Date: model.Date{
						Year: 2014,
					},
				},
				Content: &model.Content{
					Type:   model.ContentTypeMovie,
					Source: "tmdb",
					ID:     "118340",
					Title:  "Guardians of the Galaxy",
					ReleaseDate: model.Date{
						Year:  2014,
						Month: 7,
						Day:   30,
					},
					ReleaseYear:   2014,
					Adult:         model.NewNullBool(false),
					OriginalTitle: model.NewNullString("Guardians of the Galaxy"),
					Popularity:    model.NewNullFloat32(0),
					VoteAverage:   model.NewNullFloat32(0),
					VoteCount:     model.NewNullUint(0),
				},
			},
		},
		{
			torrent: model.Torrent{
				Name:        "The XXX Movie 1080p.mkv",
				FilesStatus: model.FilesStatusSingle,
				Extension:   model.NewNullString("mkv"),
				Size:        1000000000,
			},
			expected: classification.Result{
				ContentAttributes: classification.ContentAttributes{
					ContentType:     model.NewNullContentType(model.ContentTypeXxx),
					VideoResolution: model.NewNullVideoResolution(model.VideoResolutionV1080p),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("torrent: %s", tc.torrent.Name), func(t *testing.T) {
			t.Parallel()

			mocks := newTestClassifierMocks(t)

			source, sourceErr := yamlSourceProvider{rawSourceProvider: coreSourceProvider{}}.source()
			if sourceErr != nil {
				t.Fatal(sourceErr)
				return
			}

			workflow, compileErr := mocks.compiler.Compile(source)
			if compileErr != nil {
				t.Fatal(compileErr)
				return
			}

			if tc.prepareMocks != nil {
				tc.prepareMocks(mocks)
			}

			result, runErr := workflow.Run(context.Background(), "default", tc.flags, tc.torrent)
			if runErr != nil {
				assert.Equal(t, tc.expectedErr, runErr)
				t.Log(runErr)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

var (
	frierenSearchResult = tmdb.SearchTvResult{
		ID:           209867,
		Name:         "Frieren: Beyond Journey's End",
		OriginalName: "葬送のフリーレン",
	}
	frierenDetails = tmdb.TvDetailsResponse{
		ID:           209867,
		Name:         "Frieren: Beyond Journey's End",
		OriginalName: "葬送のフリーレン",
		FirstAirDate: "2023-09-29",
	}
	frierenDetailsWithAlternativeTitles = func() tmdb.TvDetailsResponse {
		d := frierenDetails
		d.AlternativeTitles = tmdb.TvAlternativeTitles{
			Results: []tmdb.AlternativeTitle{
				{Iso3166_1: "US", Title: "Frieren"},
				{Iso3166_1: "JP", Title: "Sousou no Frieren"},
			},
		}

		return d
	}()
	frierenContent = model.Content{
		Type:   model.ContentTypeTvShow,
		Source: "tmdb",
		ID:     "209867",
		Title:  "Frieren: Beyond Journey's End",
		ReleaseDate: model.Date{
			Year:  2023,
			Month: 9,
			Day:   29,
		},
		ReleaseYear:   2023,
		OriginalTitle: model.NewNullString("葬送のフリーレン"),
		Popularity:    model.NewNullFloat32(0),
		VoteAverage:   model.NewNullFloat32(0),
		VoteCount:     model.NewNullUint(0),
		Collections:   []model.ContentCollection{},
	}
)

type testClassifierMocks struct {
	compiler   Compiler
	search     *classifier_mocks.LocalSearch
	tmdbClient *tmdb_mocks.Client
}

func newTestClassifierMocks(t *testing.T) testClassifierMocks {
	t.Helper()

	search := classifier_mocks.NewLocalSearch(t)
	tmdbClient := tmdb_mocks.NewClient(t)

	return testClassifierMocks{
		compiler: compiler{
			options: []compilerOption{
				compilerFeatures(defaultFeatures),
				celEnvOption,
			},
			dependencies: dependencies{
				search:     search,
				tmdbClient: tmdbClient,
			},
		},
		search:     search,
		tmdbClient: tmdbClient,
	}
}
