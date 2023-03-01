package server

import (
	"bastion/internal/auth"
	"context"
	"encoding/json"
	"errors"
	"github.com/coreos/go-oidc"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// OIDCMiddleware осуществляет авторизацию пользователя по ID Token в web-сессии (для кожаных ублюдков) или
// по Access Token в заголовке запроса (вива ла роботолюция!)
// noinspection GoNilness
func (app *BastionServer) OIDCMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		rl := ctx.Get(requestLoggerContextKey).(*zap.Logger)
		pv, err := app.accessTokenPresentAndValid(ctx)
		if err != nil {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		if pv {
			return next(ctx)
		}
		rt, err := app.rawIDToken(ctx)
		if err != nil {
			rl.Warn(err.Error())
			return app.redirectToAuthorizationServer(ctx)
		}
		idToken, err := app.oidcClient.VerifyIDToken(rt)
		if err != nil {
			rl.Warn(err.Error())
			return app.redirectToAuthorizationServer(ctx)
		}
		err = app.setClaimsInContext(idToken, ctx)
		if err != nil {
			rl.Error(err.Error())
			return ctx.NoContent(http.StatusInternalServerError)
		}
		return next(ctx)
	}
}

// redirectToAuthorizationServer создаёт новый state-токен в текущей web-сессии и перенаправляет клиента
// на сервер авторизации с созданным токеном
func (app *BastionServer) redirectToAuthorizationServer(ctx echo.Context) error {
	rl := ctx.Get(requestLoggerContextKey).(*zap.Logger)
	stateToken, err := app.newStateToken(ctx)
	if err != nil {
		rl.Error(err.Error())
		return ctx.NoContent(http.StatusInternalServerError)
	}
	return ctx.Redirect(http.StatusFound, app.oidcClient.AuthCodeURL(stateToken))
}

// setClaimsInContext сохраняет клеймы из idToken в контекст текущего запроса
func (app *BastionServer) setClaimsInContext(idToken *oidc.IDToken, ctx echo.Context) error {
	rl := ctx.Get(requestLoggerContextKey).(*zap.Logger)
	var claims *auth.UserClaims
	err := idToken.Claims(&claims)
	if err != nil {
		rl.Error(err.Error())
		return err
	}
	ctx.Set("DisplayName", claims.DisplayName)
	ctx.Set("SID", claims.SID)
	ctx.Set("Email", claims.Email)
	return nil
}

// accessTokenPresentAndValid возвращает true, если токен доступа присутствует в запросе, иначе false
// Если токен присутствует, но не является валидным, то возвращается true и ошибка 'unauthorized'
func (app *BastionServer) accessTokenPresentAndValid(ctx echo.Context) (bool, error) {
	rl := ctx.Get(requestLoggerContextKey).(*zap.Logger)
	authHeader := ctx.Request().Header["Authorization"]
	if len(authHeader) != 1 {
		return false, nil
	}
	s := strings.Split(authHeader[0], " ")
	tokenType := s[0]
	if tokenType != "Bearer" {
		rl.Error("Unsupported 'Authorization' header type", zap.String("token_type", tokenType))
		return true, errors.New("unauthorized")
	}
	token := s[1]

	c, f := context.WithTimeout(context.Background(), time.Second*5)
	defer f()
	payload, err := app.keySet.VerifySignature(c, token)
	if err != nil {
		rl.Error(err.Error())
		return true, errors.New("unauthorized")
	}
	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		rl.Error(err.Error())
		return true, errors.New("unauthorized")
	}
	accessToken := cast.ToStringMap(v)
	appID := cast.ToString(accessToken["appid"])
	for i, ac := range app.config.OIDC.AllowedConfidentialClientIDs {
		if ac == appID {
			break
		}
		if i == len(app.config.OIDC.AllowedConfidentialClientIDs)-1 {
			rl.Error("Client is not allowed to access this server", zap.String("client-id", appID))
			return true, errors.New("unauthorized")
		}
	}
	iat := cast.ToInt64(accessToken["iat"])
	if time.Unix(iat, 0).After(time.Now()) {
		rl.Error("Access token issued in future")
		return true, errors.New("unauthorized")
	}
	exp := cast.ToInt64(accessToken["exp"])
	if time.Unix(exp, 0).Before(time.Now()) {
		rl.Error("Access token expired")
		return true, errors.New("unauthorized")
	}
	return true, nil
}
