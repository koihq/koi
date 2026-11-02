package demoseed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiapi"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

func seedDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := koidatabase.Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, koidatabase.Migrate(db))

	return db
}

func demoContext(ctx context.Context) context.Context {
	return koiauth.WithIdentity(ctx, koiauth.Identity{
		UserID:         demoID("user"),
		OrganizationID: demoID("organization"),
		Role:           koidomain.RoleOwner,
	})
}

func TestRunScenarios(t *testing.T) {
	db := seedDatabase(t)
	// Exercise the UTC boundary: the first collection must remain on today's date.
	now := time.Date(2026, 11, 2, 0, 0, 30, 0, time.UTC)
	require.NoError(t, Run(t.Context(), db, now, false))
	store := koidatabase.NewStore(db)
	ctx := demoContext(t.Context())
	page := koidomain.PageParams{Page: 1, PageSize: 20}

	me, err := koidomain.NewAuthCommander(store).Login(t.Context(), Email, Password)
	require.NoError(t, err)
	require.Equal(t, demoID("user"), me.User.ID)
	require.Equal(t, demoID("organization"), me.Organization.ID)

	brands, err := store.BrandRepo().List(ctx, koidomain.BrandListParams{Page: page})
	require.NoError(t, err)
	require.Len(t, brands.Items, 3)
	for _, brand := range brands.Items {
		require.Equal(t, "US", brand.Country)
		require.Equal(t, repository, brand.GitHubRepo)
		require.Empty(t, brand.Domain)
		require.Nil(t, brand.NextVisibilityAt)
	}

	t.Run("populated", func(t *testing.T) {
		promptsPage, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{BrandID: demoID("history"), Page: page})
		require.NoError(t, err)
		require.EqualValues(t, 32, promptsPage.TotalItems)
		require.True(t, promptsPage.HasNext)
		history, err := store.VisibilityRepo().ListHistoricalPrompts(ctx, demoID("history"), koidomain.PromptListParams{Page: page})
		require.NoError(t, err)
		require.EqualValues(t, 2, history.TotalItems)

		var cells []koidomain.VisibilityResult
		require.NoError(t, db.Where("collector = ?", koidomain.CollectorStub).Find(&cells).Error)
		counts := map[koidomain.Outcome]int{}
		for _, cell := range cells {
			require.NoError(t, cell.Validate())
			require.NotNil(t, cell.Outcome)
			counts[*cell.Outcome]++
			require.False(t, cell.CreatedAt.After(now))
			require.Nil(t, cell.CloroTaskID)
			if *cell.Outcome != koidomain.OutcomeError {
				require.Contains(t, cell.RawAnswer, "simulated answer")
				require.Contains(t, cell.AnswerMarkdown, "simulated answer")
				require.Equal(t, *cell.Outcome, koidomain.ScoreVisibility(koidomain.BrandSnapshot{
					Name:       "Koi",
					GitHubRepo: repository,
				}, cell.RawAnswer, cell.Sources))
			}
		}
		require.Len(t, cells, 20540)
		for _, outcome := range []koidomain.Outcome{koidomain.OutcomeCited, koidomain.OutcomeMentioned, koidomain.OutcomeMiss, koidomain.OutcomeError} {
			require.Positive(t, counts[outcome], "missing %s", outcome)
		}

		ids := []uuid.UUID{demoID("history/prompt/0"), demoID("history/prompt/1"), demoID("history/prompt/2"), demoID("history/prompt/3")}
		for _, days := range []int{7, 30, 90} {
			byPrompt, report, err := store.VisibilityRepo().AggregateVisibility(ctx, demoID("history"), ids, days, nil, now)
			require.NoError(t, err)
			require.Len(t, report.Series, days)
			require.Len(t, report.Engines, 4)
			require.Len(t, report.Competitors, 6)
			require.Positive(t, report.Metric.Completed)
			require.Positive(t, report.Previous.Completed)
			require.Equal(t, 100.0, *byPrompt[ids[0]].Metric.Rate)
			require.Equal(t, 0.0, *byPrompt[ids[1]].Metric.Rate)
			if days >= 30 {
				require.Greater(t, *byPrompt[ids[2]].Metric.Rate, *byPrompt[ids[2]].Previous.Rate)
				require.Less(t, *byPrompt[ids[3]].Metric.Rate, *byPrompt[ids[3]].Previous.Rate)
			}
			for _, competitor := range report.Competitors {
				require.Positive(t, competitor.Metric.Cited+competitor.Metric.Mentioned)
			}
		}

		_, report, err := store.VisibilityRepo().AggregateVisibility(ctx, demoID("history"), nil, 30, []koidomain.EngineID{koidomain.EnginePerplexity}, now)
		require.NoError(t, err)
		require.Len(t, report.Engines, 1)
		require.Contains(t, report.Engines, koidomain.EnginePerplexity)
	})

	t.Run("empty", func(t *testing.T) {
		promptsPage, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{BrandID: demoID("empty"), Page: page})
		require.NoError(t, err)
		require.Equal(t, &koidomain.Page[koidomain.Prompt]{Items: []koidomain.Prompt{}, CurrentPage: 1}, promptsPage)
		competitorsPage, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{BrandID: demoID("empty"), Page: page})
		require.NoError(t, err)
		require.Equal(t, &koidomain.Page[koidomain.Competitor]{Items: []koidomain.Competitor{}, CurrentPage: 1}, competitorsPage)
		runs, err := store.VisibilityRepo().List(ctx, demoID("empty"), page)
		require.NoError(t, err)
		require.Equal(t, &koidomain.Page[koidomain.VisibilityRun]{Items: []koidomain.VisibilityRun{}, CurrentPage: 1}, runs)
		tracking, err := store.VisibilityRepo().Tracking(ctx, demoID("empty"))
		require.NoError(t, err)
		require.False(t, tracking.PromptsReviewed)
		require.False(t, tracking.CompetitorsReviewed)
		require.Nil(t, tracking.StartedAt)
	})

	t.Run("first collection", func(t *testing.T) {
		runs, err := store.VisibilityRepo().List(ctx, demoID("first"), page)
		require.NoError(t, err)
		require.Len(t, runs.Items, 1)
		run := runs.Items[0]
		require.Equal(t, koidomain.VisibilitySucceeded, run.State)
		require.EqualValues(t, 20, run.CompletedResults)
		require.Equal(t, "Koi", run.BrandSnapshot.Name)
		require.Equal(t, "US", run.BrandSnapshot.Country)
		require.Equal(t, now.Format(time.DateOnly), run.CreatedAt.UTC().Format(time.DateOnly))
		results, err := store.VisibilityRepo().ListResults(ctx, run.ID, page)
		require.NoError(t, err)
		require.Len(t, results.Items, 20)
		for _, result := range results.Items {
			require.NotEmpty(t, result.Appearances)
		}
		_, report, err := store.VisibilityRepo().AggregateVisibility(ctx, demoID("first"), nil, 30, nil, now)
		require.NoError(t, err)
		require.Equal(t, 20, report.Metric.Completed)
		require.Nil(t, report.Previous.Rate)
		require.Zero(t, report.Previous.Completed)
		for _, day := range report.Series[:29] {
			require.Nil(t, day.Metric.Rate)
		}
		require.Equal(t, 20, report.Series[29].Metric.Completed)
	})

	t.Run("offline snapshots", func(t *testing.T) {
		var eligible int64
		require.NoError(t, db.Model(&koidomain.Brand{}).Where(`next_visibility_at <= ? OR
			(visibility_started_at IS NULL AND prompts_reviewed_at IS NOT NULL AND competitors_reviewed_at IS NOT NULL)`, now.AddDate(1, 0, 0)).Count(&eligible).Error)
		require.Zero(t, eligible)
		var runs []koidomain.VisibilityRun
		require.NoError(t, db.Find(&runs).Error)
		require.Len(t, runs, 181)
		for _, run := range runs {
			require.Equal(t, "US", run.BrandSnapshot.Country)
			require.Equal(t, koidomain.VisibilitySucceeded, run.State)
			require.Nil(t, run.RiverJobID)
			require.NotNil(t, run.FinishedAt)
		}
		// Seeding does not even require River's tables to exist.
		require.False(t, db.Migrator().HasTable("river_job"))
	})

	t.Run("cookie login", func(t *testing.T) {
		sess := scs.New()
		sess.Cookie.Name = "koi_session"
		server := httptest.NewServer(koiapi.NewServer(store, sess, nil, false))
		t.Cleanup(server.Close)
		jar, err := cookiejar.New(nil)
		require.NoError(t, err)
		client := &http.Client{Jar: jar}
		body := fmt.Sprintf(`{"email":%q,"password":%q}`, Email, Password)
		response, err := client.Post(server.URL+"/api/v1/auth/login", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.NoError(t, response.Body.Close())
		require.NotEmpty(t, response.Cookies())
		require.Equal(t, "koi_session", response.Cookies()[0].Name)
		response, err = client.Get(server.URL + "/api/v1/brands")
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		var brands koidomain.Page[koiapi.BrandRes]
		require.NoError(t, json.NewDecoder(response.Body).Decode(&brands))
		require.Len(t, brands.Items, 3)
	})

	t.Run("preserve and reset", func(t *testing.T) {
		foreign, err := koidomain.NewAuthCommander(store).Signup(t.Context(), koidomain.SignupParams{Email: "other@example.com", Password: "another-password"})
		require.NoError(t, err)
		foreignCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: foreign.User.ID, OrganizationID: foreign.Organization.ID, Role: koidomain.RoleOwner})
		foreignBrand, err := koidomain.NewBrandCommander(store).Create(foreignCtx, koidomain.BrandParams{Name: "Other brand", Domain: "example.com", Country: "US"})
		require.NoError(t, err)
		extraBrand, err := koidomain.NewBrandCommander(store).Create(ctx, koidomain.BrandParams{Name: "My extra brand", Domain: "example.org", Country: "US"})
		require.NoError(t, err)
		_, err = store.BrandRepo().Get(foreignCtx, demoID("history"))
		require.ErrorIs(t, err, koidomain.ErrNotFound)

		require.NoError(t, db.Model(&koidomain.Brand{}).Where("id = ?", demoID("history")).Update("name", "Edited demo").Error)
		require.NoError(t, Run(t.Context(), db, now.AddDate(0, 0, 1), false))
		brand, err := store.BrandRepo().Get(ctx, demoID("history"))
		require.NoError(t, err)
		require.Equal(t, "Edited demo", brand.Name)
		var resultCount int64
		require.NoError(t, db.Model(&koidomain.VisibilityResult{}).Count(&resultCount).Error)
		require.EqualValues(t, 20540, resultCount)

		// Recreate a missing scenario without changing the existing ones.
		require.NoError(t, db.Delete(&koidomain.Brand{}, "id = ?", demoID("empty")).Error)
		require.NoError(t, Run(t.Context(), db, now, false))
		require.NoError(t, Run(t.Context(), db, now.AddDate(0, 0, 2), true))
		brand, err = store.BrandRepo().Get(ctx, demoID("history"))
		require.NoError(t, err)
		require.Equal(t, "Koi · Storico", brand.Name)
		run, err := store.VisibilityRepo().Get(ctx, demoID("first/run/0"))
		require.NoError(t, err)
		require.Equal(t, now.AddDate(0, 0, 2), run.FinishedAt.UTC())
		_, err = store.BrandRepo().Get(ctx, extraBrand.ID)
		require.NoError(t, err)
		_, err = store.BrandRepo().Get(foreignCtx, foreignBrand.ID)
		require.NoError(t, err)
		require.NoError(t, db.Model(&koidomain.VisibilityResult{}).Count(&resultCount).Error)
		require.EqualValues(t, 20540, resultCount)
		_, err = koidomain.NewAuthCommander(store).Login(t.Context(), Email, Password)
		require.NoError(t, err)
	})
}

func TestRunRejectsOccupiedEmail(t *testing.T) {
	db := seedDatabase(t)
	user := koidomain.User{Email: Email, PasswordHash: "existing"}
	require.NoError(t, db.Create(&user).Error)
	require.ErrorContains(t, Run(t.Context(), db, time.Now(), false), "occupied")
	var count int64
	require.NoError(t, db.Model(&koidomain.Organization{}).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.First(&user, "id = ?", user.ID).Error)
	require.Equal(t, "existing", user.PasswordHash)
}

func TestRunRollsBackAccountAndReset(t *testing.T) {
	db := seedDatabase(t)
	// The intentionally rejected batch would otherwise log thousands of SQL placeholders.
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	// Force failure after the account and brands have already been written.
	require.NoError(t, db.Exec(`ALTER TABLE visibility_results ADD CONSTRAINT reject_demo CHECK (collector <> 'stub')`).Error)
	require.Error(t, Run(t.Context(), db, now, false))
	for _, model := range []any{&koidomain.User{}, &koidomain.Organization{}, &koidomain.Membership{}, &koidomain.Brand{}, &koidomain.Prompt{}, &koidomain.Competitor{}, &koidomain.VisibilityRun{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.Zero(t, count)
	}
	require.NoError(t, db.Exec(`ALTER TABLE visibility_results DROP CONSTRAINT reject_demo`).Error)
	require.NoError(t, Run(t.Context(), db, now, false))

	// A reset failure must restore the previous demo, including its deleted rows.
	require.NoError(t, db.Exec(`ALTER TABLE visibility_results ADD CONSTRAINT reject_demo CHECK (collector <> 'stub') NOT VALID`).Error)
	require.Error(t, Run(t.Context(), db, now.AddDate(0, 0, 1), true))
	var count int64
	require.NoError(t, db.Model(&koidomain.VisibilityResult{}).Count(&count).Error)
	require.EqualValues(t, 20540, count)
	var run koidomain.VisibilityRun
	require.NoError(t, db.First(&run, "id = ?", demoID("first/run/0")).Error)
	require.Equal(t, now, run.FinishedAt.UTC())
}
