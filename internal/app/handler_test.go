package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/auth"
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/config"
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/store"
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/models"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func Test_ShortenAPIHandler(t *testing.T) {
	targetURL := "https://praktikum.yandex.ru/"

	instance := &Instance{
		baseURL: "http://localhost:8080",
		store:   store.NewInMemory(),
	}

	testCases := []struct {
		name             string
		url              string
		expectedStatus   int
		expectedResponse []byte
	}{
		{
			name:             "bad_request",
			url:              "htt_p://o.com",
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: []byte("Cannot parse given string as URL"),
		},
		{
			name:             "success",
			url:              targetURL,
			expectedStatus:   http.StatusCreated,
			expectedResponse: []byte("{\"result\":\"http://localhost:8080/0\"}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(models.ShortenRequest{URL: tc.url})
			require.NoError(t, err)
			body := bytes.NewBuffer(b)

			r := httptest.NewRequest("POST", "http://localhost:8080/api/shorten", body)
			w := httptest.NewRecorder()

			instance.ShortenAPIHandler(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.Bytes())
		})
	}
}

func Test_expander(t *testing.T) {
	expectedURL := "https://praktikum.yandex.ru/"
	parsedURL, _ := url.Parse(expectedURL)

	storage := store.NewInMemory()
	id, _ := storage.Save(context.Background(), parsedURL)

	instance := &Instance{
		baseURL: "http://localhost:8080",
		store:   storage,
	}

	testCases := []struct {
		name             string
		id               string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "bad_request",
			id:               "",
			expectedStatus:   http.StatusBadRequest,
			expectedLocation: "",
		},
		{
			name:             "not_found",
			id:               "-1",
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		{
			name:             "success",
			id:               id,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: expectedURL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://localhost:8080/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			instance.ExpandHandler(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, tc.expectedLocation, w.Header().Get("Location"))
		})
	}
}

func Test_userURLs(t *testing.T) {
	uid := uuid.Must(uuid.NewV4())
	u, _ := url.Parse("https://praktikum.yandex.ru/")

	storage := store.NewInMemory()
	id, _ := storage.SaveUser(context.Background(), uid, u)

	instance := &Instance{
		baseURL: "http://localhost:8080",
		store:   storage,
	}

	testCases := []struct {
		name           string
		ctx            context.Context
		expectedStatus int
		expectedBody   []byte
	}{
		{
			name:           "no_uid",
			ctx:            context.Background(),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   nil,
		},
		{
			name:           "no_urls",
			ctx:            auth.Context(context.Background(), uuid.Must(uuid.NewV4())),
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:           "has_urls",
			ctx:            auth.Context(context.Background(), uid),
			expectedStatus: http.StatusOK,
			expectedBody:   []byte("[{\"short_url\":\"http://localhost:8080/" + id + "\",\"original_url\":\"https://praktikum.yandex.ru/\"}]\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://localhost:8080/user/urls", nil)
			r = r.WithContext(tc.ctx)

			w := httptest.NewRecorder()
			instance.UserURLsHandler(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, tc.expectedBody, w.Body.Bytes())
		})
	}
}

func randStringBytes() string {

	b := make([]byte, letterCount)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

const (
	letterBytes = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterCount = 8
)

func BenchmarkShortener(b *testing.B) {

	rand.Seed(time.Now().UnixNano())

	storage := store.NewInMemory()
	defer storage.Close()

	instance := NewInstance(config.BaseURL, storage)

	b.ResetTimer()

	b.Run("check", func(b *testing.B) {
		for i := 0; i < b.N; i++ {

			b.StopTimer()

			rawUrl := fmt.Sprintf("https://%s.com", randStringBytes())
			u, _ := url.Parse(rawUrl)

			b.StartTimer()

			_, _ = instance.shorten(context.Background(), u)
		}
	})

}
