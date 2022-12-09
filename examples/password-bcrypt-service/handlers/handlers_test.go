package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/illyasch/worker-pool/examples/password-bcrypt-service/handlers"
	"github.com/illyasch/worker-pool/pool"
)

var (
	stdLgr *log.Logger
)

type response struct {
	Error string `json:"error,omitempty"`
	Hash  string `json:"hash"`
}

func TestMain(m *testing.M) {
	stdLgr = log.New(os.Stdout, "test", log.LstdFlags)
	os.Exit(m.Run())
}

func TestMakeHandler(t *testing.T) {
	t.Run(`bcrypt a password`, func(t *testing.T) {
		t.Parallel()
		const pwd = "qwertyuuiiopasdfg1233456969"

		workers := pool.NewNonBlocking[string](runtime.NumCPU())
		workers.Run(context.Background())
		cfg := handlers.APIConfig{
			BusyTimeout:    1000 * time.Millisecond,
			Log:            stdLgr,
			Workers:        workers,
			PasswordMinLen: 8,
		}

		vals := url.Values{}
		vals.Set("password", pwd)
		req := httptest.NewRequest(http.MethodPost, "/bcrypt", strings.NewReader(vals.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		cfg.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := response{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Empty(t, resp.Error)

		err = bcrypt.CompareHashAndPassword([]byte(resp.Hash), []byte(pwd))
		require.NoError(t, err)
	})

	t.Run(`minimal load test`, func(t *testing.T) {
		t.Parallel()
		workers := pool.NewNonBlocking[string](runtime.NumCPU())
		workers.Run(context.Background())
		cfg := handlers.APIConfig{
			BusyTimeout:    66 * time.Millisecond,
			Log:            stdLgr,
			Workers:        workers,
			PasswordMinLen: 8,
		}

		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("HTTP request %d", i+1), func(t *testing.T) {
				t.Parallel()
				pwd, err := uuid.NewUUID()
				require.NoError(t, err)

				vals := url.Values{}
				vals.Set("password", pwd.String())
				req := httptest.NewRequest(http.MethodPost, "/bcrypt", strings.NewReader(vals.Encode()))
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

				w := httptest.NewRecorder()
				cfg.Router().ServeHTTP(w, req)

				assert.NotEqual(t, http.StatusInternalServerError, w.Code)
				t.Logf("return status %d", w.Code)
			})
		}
	})

	t.Run(`incorrect password`, func(t *testing.T) {
		t.Parallel()
		workers := pool.NewNonBlocking[string](runtime.NumCPU())
		workers.Run(context.Background())
		cfg := handlers.APIConfig{
			BusyTimeout:    100 * time.Millisecond,
			Log:            stdLgr,
			Workers:        workers,
			PasswordMinLen: 8,
		}

		vals := url.Values{}
		vals.Set("password", "qwerty")
		req := httptest.NewRequest(http.MethodPost, "/bcrypt", strings.NewReader(vals.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		cfg.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		resp := response{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, "input password is incorrect", resp.Error)
	})

	t.Run(`busy timeout`, func(t *testing.T) {
		t.Parallel()
		workers := pool.NewNonBlocking[string](runtime.NumCPU())
		workers.Run(context.Background())
		cfg := handlers.APIConfig{
			BusyTimeout:    0,
			Log:            stdLgr,
			Workers:        workers,
			PasswordMinLen: 8,
		}

		vals := url.Values{}
		vals.Set("password", "qwertyegegrggeeggre")
		req := httptest.NewRequest(http.MethodPost, "/bcrypt", strings.NewReader(vals.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		cfg.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		resp := response{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, "Too Many Requests", resp.Error)
	})
}
