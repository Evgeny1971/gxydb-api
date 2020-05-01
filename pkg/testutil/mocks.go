package testutil

import (
	"context"

	"github.com/coreos/go-oidc"
	"github.com/stretchr/testify/mock"
)

type OIDCTokenVerifier struct {
	mock.Mock
}

func (m *OIDCTokenVerifier) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	args := m.Called(ctx, rawIDToken)
	return args.Get(0).(*oidc.IDToken), args.Error(1)
}
