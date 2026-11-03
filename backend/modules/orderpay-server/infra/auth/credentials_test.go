package auth

import (
	"context"
	"testing"

	"github.com/fino-io/gokit/session"
	"github.com/stretchr/testify/require"
)

func TestLoadRejectsMissingWeakAndUnscopedCredentials(t *testing.T) {
	for _, value := range []string{"", "{}", `{"short":{"merchantId":"019a4a70-0000-7000-8000-000000000001"}}`, `{"a-private-secret-at-least-thirty-two-characters":{"merchantId":"not-a-uuid"}}`} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ORDERPAY_API_KEYS", value)
			_, err := Load()
			require.Error(t, err)
		})
	}
}

func TestCredentialsResolveOnlyConfiguredKeys(t *testing.T) {
	const key = "a-private-secret-at-least-thirty-two-characters"
	t.Setenv("ORDERPAY_API_KEYS", `{"`+key+`":{"merchantId":"019a4a70-0000-7000-8000-000000000001"}}`)
	credentials, err := Load()
	require.NoError(t, err)
	_, ok := credentials.Authenticate(context.Background())
	require.False(t, ok)
	_, ok = credentials.Authenticate(session.WithToken(context.Background(), "unknown"))
	require.False(t, ok)
	principal, ok := credentials.Authenticate(session.WithToken(context.Background(), key))
	require.True(t, ok)
	require.Equal(t, "019a4a70-0000-7000-8000-000000000001", principal.MerchantID)
	require.False(t, principal.Admin)
}
