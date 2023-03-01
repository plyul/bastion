package proxy

import (
	"bastion/internal/api/client"
	"bastion/internal/log"
	"fmt"

	"github.com/gliderlabs/ssh"
	"go.uber.org/zap"
)

type BastionProxy struct {
	config    ConfigStruct
	logger    *zap.Logger
	apiClient client.APIClient
}

func New() (*BastionProxy, error) {
	var proxy BastionProxy
	var err error
	proxy.config = LoadConfiguration()
	proxy.logger, err = log.Init(proxy.config.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("error initializing logging: %s", err)
	}
	c := client.APIClientConfig{
		Endpoint:         proxy.config.API.URL,
		CertificateFile:  proxy.config.API.CertificateFile,
		OIDCIssuer:       proxy.config.OIDC.Issuer,
		OIDCClientID:     proxy.config.OIDC.ClientID,
		OIDCClientSecret: proxy.config.OIDC.ClientSecret,
		Logger:           proxy.logger,
	}
	proxy.apiClient, err = client.New(c)
	if err != nil {
		proxy.logger.Error(err.Error())
		return nil, err
	}
	return &proxy, nil
}

func (app *BastionProxy) Run() {
	app.logger.Info("Bastion proxy listening", zap.String("address", app.config.BindAddress))
	app.logger.Fatal(ssh.ListenAndServe(app.config.BindAddress, app.SessionHandler, ssh.WrapConn(app.ConnCallback)).Error())
}

func (app *BastionProxy) Shutdown() {
	app.logger.Info("Shutdown")
	_ = app.logger.Sync()
}
