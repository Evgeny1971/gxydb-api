package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/Bnei-Baruch/gxydb-api/instrumentation"
)

var requestLog = zerolog.New(os.Stdout).With().Timestamp().Caller().Stack().Logger()

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	zerolog.CallerFieldName = "line"
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		rel := strings.Split(file, "gxydb-api/")
		return fmt.Sprintf("%s:%d", rel[1], line)
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.With().Stack()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	h1 := hlog.NewHandler(requestLog)
	h2 := hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		path := r.URL.EscapedPath()

		event := hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("path", path).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration)

		if rCtx, ok := ContextFromRequest(r); ok {
			event.Str("ip", rCtx.IP)
			if rCtx.IDClaims != nil {
				event.Str("user", rCtx.IDClaims.Sub)
			}
			if status >= http.StatusBadRequest {
				event.Interface("params", rCtx.Params)
			}

			// some middleware respond without ever reaching the router (OPTIONS, 401,403, etc...)
			if rCtx.RouteName != "" {
				path = rCtx.RouteName
			} else {
				path = "any"
			}
		}

		event.Msg("")

		instrumentation.Stats.RequestDurationHistogram.
			WithLabelValues(r.Method, path, strconv.Itoa(status)).
			Observe(duration.Seconds())
	})
	h3 := hlog.RequestIDHandler("request_id", "X-Request-ID")
	return h1(h2(h3(next)))
}
