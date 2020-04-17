package server

import (
	"encoding/hex"
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

const cookieName = "bastionData"

func (app *BastionServer) initSessionStore() {
	store := sessions.NewFilesystemStore(app.config.Web.SessionsDir, securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))
	store.MaxLength(0)
	store.MaxAge(app.config.OIDC.SessionTTLSeconds)
	app.sessions = store
}

//noinspection GoNilness
func (app *BastionServer) saveTokenToSession(token *oauth2.Token, context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	session, err := app.sessions.Get(context.Request(), cookieName)
	if err != nil {
		rl.Warn(err.Error())
	}
	session.Values["accessToken"] = token.AccessToken
	session.Values["accessTokenExpiry"] = token.Expiry.String()
	session.Values["accessTokenType"] = token.TokenType
	session.Values["refreshToken"] = token.RefreshToken
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		rl.Error("No id_token in response")
		idToken = ""
	}
	session.Values["idToken"] = idToken
	err = session.Save(context.Request(), context.Response().Writer)
	if err != nil {
		rl.Error(err.Error())
		return err
	}
	return nil
}

//noinspection GoNilness
func (app *BastionServer) newStateToken(context echo.Context) (string, error) {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	rdata := securecookie.GenerateRandomKey(64)
	if rdata == nil {
		return "", errors.New("error generating random bits")
	}
	stateToken := hex.EncodeToString(rdata)
	session, _ := app.sessions.Get(context.Request(), cookieName)
	session.Values["stateToken"] = stateToken
	err := session.Save(context.Request(), context.Response().Writer)
	if err != nil {
		rl.Error(err.Error())
		return stateToken, err
	}
	return stateToken, nil
}

//noinspection GoNilness
func (app *BastionServer) rawIDToken(context echo.Context) (string, error) {
	session, err := app.sessions.Get(context.Request(), cookieName)
	if err != nil {
		return "", err
	}
	rawIDToken, ok := session.Values["idToken"].(string)
	if !ok {
		return "", err
	}
	return rawIDToken, nil
}
