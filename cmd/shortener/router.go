package main

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid"

	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/app"
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/auth"
)

func newRouter(i *app.Instance) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(gzipMiddleware, authMiddleware)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/", i.ShortenHandler)
	r.Post("/api/shorten", i.ShortenAPIHandler)
	r.Post("/api/shorten/batch", i.BatchShortenAPIHandler)
	r.Delete("/api/user/urls", i.BatchRemoveAPIHandler)
	r.Get("/{id}", i.ExpandHandler)
	r.Get("/api/user/urls", i.UserURLsHandler)
	r.Get("/ping", i.PingHandler)

	r.Get("/debug/pprof/", pprof.Index)
	r.Get("/debug/pprof/cmdline", pprof.Cmdline)
	r.Get("/debug/pprof/profile", pprof.Profile)
	r.Get("/debug/pprof/symbol", pprof.Symbol)
	r.Get("/debug/pprof/trace", pprof.Trace)
	r.Get("/debug/pprof/{cmd}", pprof.Index)

	return r
}

func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := newCompressWriter(w)
			defer cw.Close()

			w.Header().Set("Content-Encoding", "gzip")
			ow = cw
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uid *uuid.UUID

		cookie, err := r.Cookie("auth")
		if cookie != nil {
			uid, err = auth.DecodeUIDFromHex(cookie.Value)
		}
		// generate new uid if failed to obtain existing
		if uid == nil {
			userID := ensureRandom()
			uid = &userID
		}

		// set new auth cookie in case of absence or decode error
		if err != nil {
			value, err := auth.EncodeUIDToHex(*uid)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("cannot encode auth cookie"))
				return
			}
			cookie = &http.Cookie{Name: "auth", Value: value}
			http.SetCookie(w, cookie)
		}

		// set uid to context
		ctx := auth.Context(r.Context(), *uid)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func ensureRandom() (res uuid.UUID) {
	for i := 0; i < 10; i++ {
		res = uuid.Must(uuid.NewV4())
	}
	return
}
