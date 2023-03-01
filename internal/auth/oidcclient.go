package auth

import (
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type OIDCClient struct {
	provider  oidc.Provider
	oidc      oidc.Config
	acgConfig oauth2.Config            // Config for 'Authorization Code Grant'
	ccgConfig clientcredentials.Config // Config for 'Client Credentials Grant'
}

const contextTimeout = time.Second * 15

func New(issuer, clientID, clientSecret, redirectURL string, scopes []string) (*OIDCClient, error) {
	c := &OIDCClient{
		provider:  oidc.Provider{},
		oidc:      oidc.Config{ClientID: clientID},
		acgConfig: oauth2.Config{},
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return c, err
	}
	c.provider = *provider

	s := scopes
	s = append(s, oidc.ScopeOpenID)
	c.acgConfig = oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       s,
	}
	c.ccgConfig = clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     provider.Endpoint().TokenURL,
	}
	return c, nil
}

func (client *OIDCClient) HTTPClientWithAccessToken(ctx context.Context) *http.Client {
	return client.ccgConfig.Client(ctx)
}

func (client *OIDCClient) AuthCodeURL(state string) string {
	return client.acgConfig.AuthCodeURL(state)
}

func (client *OIDCClient) FetchToken(code string, logger *zap.Logger) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	token, err := client.acgConfig.Exchange(ctx, code)
	if err != nil {
		logger.Error(err.Error())
		return token, err
	}
	return token, nil
}

func (client *OIDCClient) VerifyIDToken(rawIDToken string) (*oidc.IDToken, error) {
	verifier := client.provider.Verifier(&client.oidc)
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	return idToken, nil
}
