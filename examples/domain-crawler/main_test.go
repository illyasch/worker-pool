package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_measureDomainResponse(t *testing.T) {
	t.Run("10 domains success", func(t *testing.T) {
		inp := ""
		for i := 0; i < 10; i++ {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				w.Write([]byte("{}"))
			}))
			defer func(srv *httptest.Server) {
				srv.Close()
			}(s)

			inp = fmt.Sprintf("%s%s\n", inp, s.URL)
		}

		got := measureDomainResponse(strings.NewReader(inp), "https", 10, 1)
		assert.Equal(t, got.num, 10)
	})
}
