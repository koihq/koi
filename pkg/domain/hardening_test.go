package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func fieldCode(t *testing.T, err error) string {
	t.Helper()

	var domainErr *Error
	require.ErrorAs(t, err, &domainErr)
	require.Len(t, domainErr.Fields, 1)

	return domainErr.Fields[0].Code
}

func TestValidatePassword(t *testing.T) {
	require.Equal(t, FieldMinLength, fieldCode(t, validatePassword(strings.Repeat("a", 11))))
	require.NoError(t, validatePassword(strings.Repeat("a", 12)))
	require.NoError(t, validatePassword(strings.Repeat("è", 12)), "counts characters, not bytes")
	require.NoError(t, validatePassword(strings.Repeat("a", 256)))
	require.Equal(t, FieldMaxLength, fieldCode(t, validatePassword(strings.Repeat("a", 257))))
}

func TestNewPasswordsMeetTheOWASPConfiguration(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	require.NoError(t, err)
	require.Contains(t, hash, "m=19456$t=2$")
	require.True(t, CheckPassword(hash, "correct-horse-battery"))
}

func TestBrandNormalizesItsSite(t *testing.T) {
	brand, err := NewBrand(uuid.New(), CreateBrandParams{
		Name:    "Acme",
		Domain:  " WWW.Acme.com ",
		Country: "US",
	})

	require.NoError(t, err)
	require.Equal(t, "acme.com", brand.Domain)
}

func TestBrandRejectsAHostWithoutADot(t *testing.T) {
	_, err := NewBrand(uuid.New(), CreateBrandParams{Name: "Acme", Domain: "intranet", Country: "US"})

	require.ErrorIs(t, err, ErrValidation)
}

func TestBrandAndPromptLengths(t *testing.T) {
	organizationID := uuid.New()
	params := func(name, description string) CreateBrandParams {
		return CreateBrandParams{Name: name, Domain: "acme.com", Country: "US", Description: description}
	}

	_, err := NewBrand(organizationID, params(strings.Repeat("n", maxBrandNameLen), strings.Repeat("d", maxBrandDescriptionLen)))
	require.NoError(t, err)

	_, err = NewBrand(organizationID, params(strings.Repeat("n", maxBrandNameLen+1), ""))
	require.Equal(t, FieldMaxLength, fieldCode(t, err))

	_, err = NewBrand(organizationID, params("Acme", strings.Repeat("d", maxBrandDescriptionLen+1)))
	require.Equal(t, FieldMaxLength, fieldCode(t, err))

	_, err = NewPrompt(uuid.New(), CreatePromptParams{Text: strings.Repeat("p", maxPromptTextLen)})
	require.NoError(t, err)

	_, err = NewPrompt(uuid.New(), CreatePromptParams{Text: strings.Repeat("p", maxPromptTextLen+1)})
	require.Equal(t, FieldMaxLength, fieldCode(t, err))
}

func TestAccountFactsDeletable(t *testing.T) {
	for _, status := range []string{"", "canceled", "incomplete_expired"} {
		require.NoError(t, AccountFacts{SubscriptionStatus: status, Members: 1}.deletable(), status)
	}

	for _, status := range []string{"active", "trialing", "past_due", "unpaid", "paused", "incomplete"} {
		require.ErrorIs(t, AccountFacts{SubscriptionStatus: status, Members: 1}.deletable(), ErrAccountSubscriptionLive, status)
	}

	require.ErrorIs(t, AccountFacts{Members: 2}.deletable(), ErrAccountHasOtherMembers)
	require.ErrorIs(t, AccountFacts{Members: 1, ActiveWork: 1}.deletable(), ErrAccountWorkActive)
}
