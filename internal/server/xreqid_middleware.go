package server

import (
	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

func (app *BastionServer) XRequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Header.Get("X-Request-ID") != "" {
			return nil
		}
		reqID := uuid.NewV4().String()
		c.Request().Header.Add("X-Request-ID", reqID)
		rl := app.logger.With(zap.String("request_id", reqID))
		c.Set(requestLoggerContextKey, rl)
		return next(c)
	}
}
