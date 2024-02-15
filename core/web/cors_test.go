package web_test

import (
	"net/http"
	"testing"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/cltest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/configtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
)

func TestCors_DefaultOrigins(t *testing.T) {
	t.Parallel()

	config := configtest.NewGeneralConfig(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.WebServer.AllowOrigins = ptr("http://localhost:3000,http://localhost:6689")
	})

	tests := []struct {
		origin     string
		statusCode int
	}{
		{"http://localhost:3000", http.StatusOK},
		{"http://localhost:6689", http.StatusOK},
		{"http://localhost:1234", http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.origin, func(t *testing.T) {
			app := cltest.NewApplicationWithConfig(t, config)

			client := app.NewHTTPClient(nil)

			headers := map[string]string{"Origin": test.origin}
			resp, cleanup := client.Get("/v2/chains/evm", headers)
			defer cleanup()
			cltest.AssertServerResponse(t, resp, test.statusCode)
		})
	}
}

func TestCors_OverrideOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		allow      string
		origin     string
		statusCode int
	}{
		{"http://erinaceus.com", "http://erinaceus.com", http.StatusOK},
		{"http://erinaceus.com", "http://localhost:3000", http.StatusForbidden},
		{"*", "http://erinaceus.com", http.StatusOK},
		{"*", "http://localhost:3000", http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.origin, func(t *testing.T) {

			config := configtest.NewGeneralConfig(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
				c.WebServer.AllowOrigins = ptr(test.allow)
			})
			app := cltest.NewApplicationWithConfig(t, config)

			client := app.NewHTTPClient(nil)

			headers := map[string]string{"Origin": test.origin}
			resp, cleanup := client.Get("/v2/chains/evm", headers)
			defer cleanup()
			cltest.AssertServerResponse(t, resp, test.statusCode)
		})
	}
}
