package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRejectsCloudBeforeOpeningDatabase(t *testing.T) {
	t.Setenv("KOI_DEPLOYMENT_MODE", "cloud")
	t.Setenv("KOI_DATABASE_URL", "invalid")
	require.ErrorContains(t, run(t.Context(), false), "selfhosted")
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	t.Setenv("KOI_DEPLOYMENT_MODE", "selfhosted")
	t.Setenv("KOI_DATABASE_URL", "")
	require.ErrorContains(t, run(t.Context(), false), "KOI_DATABASE_URL is required")
}
