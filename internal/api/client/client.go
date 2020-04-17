package client

import (
	"bastion/internal/api"
	"bastion/internal/auth"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
)

type APIClientConfig struct {
	Endpoint         string
	CertificateFile  string
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	Logger           *zap.Logger
}

type APIClient struct {
	oidcClient *auth.OIDCClient
	apiURL     string
	logger     *zap.Logger
	httpClient *http.Client
}

func New(c APIClientConfig) (APIClient, error) {
	apiClient := APIClient{}
	authClient, err := auth.New(c.OIDCIssuer, c.OIDCClientID, c.OIDCClientSecret, "", nil)
	if err != nil {
		c.Logger.Error(err.Error())
		return apiClient, err
	}
	apiClient.apiURL = c.Endpoint
	apiClient.logger = c.Logger
	apiClient.oidcClient = authClient

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		c.Logger.Error(err.Error())
		return apiClient, err
	}
	c.Logger.Debug("Loading certificate", zap.String("cert_file", c.CertificateFile))
	certs, err := ioutil.ReadFile(c.CertificateFile)
	if err != nil {
		c.Logger.Error(err.Error())
		return apiClient, err
	}
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		c.Logger.Error("appending certificate to pool failed")
		return apiClient, err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            rootCAs,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	baseHTTPClient := http.Client{
		Transport: transport,
	}
	bctx := context.Background()
	ctx := context.WithValue(bctx, oauth2.HTTPClient, &baseHTTPClient)
	client := authClient.HTTPClientWithAccessToken(ctx)

	apiClient.httpClient = client

	return apiClient, nil
}

func (a APIClient) GetSession(token string) (api.ReadSessionDTO, error) {
	sess := api.ReadSessionDTO{}
	sessionsResponse, err := a.httpClient.Get(fmt.Sprintf("%s/api/sessions/%s", a.apiURL, token))
	if err != nil {
		return sess, err
	}
	body, err := ioutil.ReadAll(sessionsResponse.Body)
	if err != nil {
		return sess, err
	}

	err = json.Unmarshal(body, &sess)
	if err != nil {
		return sess, err
	}
	err = sessionsResponse.Body.Close()
	return sess, err
}
