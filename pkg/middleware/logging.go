package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var requestLog = zerolog.New(os.Stdout).With().Timestamp().Caller().Stack().Logger()

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	zerolog.CallerFieldName = "line"
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		rel := strings.Split(file, "gxydb-api/")
		return fmt.Sprintf("%s:%d", rel[1], line)
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.With().Stack()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	h1 := hlog.NewHandler(requestLog)
	h2 := hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		event := hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("path", r.URL.EscapedPath()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration)

		if realIP, ok := RealIPFromRequest(r); ok {
			event.Str("ip", realIP)
		}

		if idClaims, ok := IDClaimsFromRequest(r); ok {
			event.Str("user", idClaims.Aud)
		}

		event.Msg("")
	})
	h3 := hlog.RequestIDHandler("request_id", "X-Request-ID")
	return h1(h2(h3(next)))
}