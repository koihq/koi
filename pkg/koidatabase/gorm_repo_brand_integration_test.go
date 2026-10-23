package koidatabase

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestBrandSearchAndActiveResolution(t *testing.T) {
	db, store, ctx, original := liveVisibilityFixture(t, false)
	created := time.Now().Add(time.Hour)
	brands := make([]koidomain.Brand, 121)
	for i := range brands {
		brands[i] = koidomain.Brand{
			BaseEntity: koidomain.BaseEntity{
				CreatedAt: created.Add(time.Duration(i) * time.Second),
			},
			OrganizationID: original.OrganizationID,
			Name:           fmt.Sprintf("Example %03d", i),
			Domain:         fmt.Sprintf("site%d.example", i),
			Country:        "IT",
		}
	}
	brands[0].Name = "100%_Koi"
	brands[1].GitHubRepo = "https://github.com/example/Project"
	require.NoError(t, db.Create(&brands).Error)

	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organizationID := testOrganizationForUser(t, store, user.ID)
	foreignCtx := testIdentity(t, store, user.ID)
	foreign := &koidomain.Brand{
		OrganizationID: organizationID,
		Name:           "100%_Koi",
		Domain:         "foreign.example",
		Country:        "IT",
	}
	require.NoError(t, store.BrandRepo().Create(foreignCtx, foreign))

	for _, query := range []string{"100%_kOI", "%_", "SITE120", "project"} {
		page, err := store.BrandRepo().List(ctx, koidomain.BrandListParams{
			Page: koidomain.PageParams{
				Page:     1,
				PageSize: 20,
			},
			Query: query,
		})
		require.NoError(t, err)
		require.EqualValues(t, 1, page.TotalItems, query)
		require.Len(t, page.Items, 1)
		require.Equal(t, original.OrganizationID, page.Items[0].OrganizationID)
		require.False(t, page.HasNext)
	}

	first, err := store.BrandRepo().List(ctx, koidomain.BrandListParams{
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 20,
		},
	})
	require.NoError(t, err)
	require.EqualValues(t, 122, first.TotalItems)
	require.Len(t, first.Items, 20)
	require.Equal(t, brands[120].ID, first.Items[0].ID)
	require.True(t, first.HasNext)

	var brandQueries []string
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("brand_resolution_queries", func(tx *gorm.DB) {
		if tx.Statement.Table == "brands" {
			brandQueries = append(brandQueries, tx.Statement.SQL.String())
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove("brand_resolution_queries") })
	querier := koidomain.NewAuthQuerier(store)
	me, err := querier.Me(ctx, original.ID)
	require.NoError(t, err)
	require.Equal(t, original.ID, me.ActiveBrand.ID)
	require.Len(t, brandQueries, 1)
	require.Contains(t, brandQueries[0], "LIMIT")
	require.NotContains(t, brandQueries[0], "count(")

	for _, preferred := range []uuid.UUID{uuid.Nil, uuid.New(), foreign.ID} {
		me, err := querier.Me(ctx, preferred)
		require.NoError(t, err)
		require.Equal(t, brands[120].ID, me.ActiveBrand.ID)
	}

	require.NoError(t, db.Where("organization_id = ?", organizationID).Delete(&koidomain.Brand{}).Error)
	me, err = querier.Me(foreignCtx, uuid.Nil)
	require.NoError(t, err)
	require.Nil(t, me.ActiveBrand)
}
