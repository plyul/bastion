package server

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func LoggerMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			req := context.Request()
			res := context.Response()

			start := time.Now()
			err := next(context)
			stop := time.Now()
			latency := stop.Sub(start)

			id := req.Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = res.Header().Get(echo.HeaderXRequestID)
			}
			idField := zap.String("request_id", id)

			status := res.Status
			herr, ok := err.(*echo.HTTPError)
			if ok {
				status = herr.Code
			}
			errorField := zap.String("error", "no error")
			if err != nil {
				errorField = zap.String("error", err.Error())
			}
			sid, sidExists := context.Get("SID").(string)
			if !sidExists {
				sid = "<NONE>"
			}
			logger.Info(http.StatusText(status),
				zap.String("user_sid", sid),
				zap.String("uri", req.RequestURI),
				zap.String("method", req.Method),
				zap.Int("status", status),
				zap.String("host", req.Host),
				zap.String("remote_ip", context.RealIP()),
				zap.String("protocol", req.Proto),
				zap.String("referer", req.Referer()),
				zap.String("user_agent", req.UserAgent()),
				zap.Int64("latency", int64(latency)),
				zap.String("latency_human", latency.String()),
				errorField,
				idField,
			)
			if !context.Response().Committed {
				return context.HTML(status, htmlLastResortResponse)
			}
			return err
		}
	}
}

const htmlLastResortResponse = "<!DOCTYPE html>" +
	"<html lang='en'><head><meta charset='UTF-8'><title>Бастион</title></head>" +
	"<body style='text-align: center'>" +
	"<div style='border: 0.2em solid black; width: 30%; padding: 1em; margin: 3em; display: inline-block;'>" +
	"<div>&mdash; 4х4 ошибочка... что случилось?</div>" +
	"<div>&mdash; По-моему, приводу пизда, ребята... может быть такое?</div>" +
	"<div>&mdash; Может быть...</div>" +
	"</div>" +
	"</body></html>"
