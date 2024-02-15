package web_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/auth"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/bridges"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/cltest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils"
	clhttptest "github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/httptest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingController_Show_APICredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start(testutils.Context(t)))

	client := app.NewHTTPClient(nil)

	resp, cleanup := client.Get("/v2/ping")
	defer cleanup()
	cltest.AssertServerResponse(t, resp, http.StatusOK)
	body := string(cltest.ParseResponseBody(t, resp))
	require.Equal(t, `{"message":"pong"}`, strings.TrimSpace(body))
}

func TestPingController_Show_ExternalInitiatorCredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start(testutils.Context(t)))

	eia := &auth.Token{
		AccessKey: "abracadabra",
		Secret:    "opensesame",
	}
	eir_url := cltest.WebURL(t, "http://localhost:8888")
	eir := &bridges.ExternalInitiatorRequest{
		Name: uuid.New().String(),
		URL:  &eir_url,
	}

	ei, err := bridges.NewExternalInitiator(eia, eir)
	require.NoError(t, err)
	err = app.BridgeORM().CreateExternalInitiator(ei)
	require.NoError(t, err)

	url := app.Server.URL + "/v2/ping"
	request, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	request.Header.Set("Content-Type", web.MediaType)
	request.Header.Set("X-Chainlink-EA-AccessKey", eia.AccessKey)
	request.Header.Set("X-Chainlink-EA-Secret", eia.Secret)

	client := clhttptest.NewTestLocalOnlyHTTPClient()
	resp, err := client.Do(request)
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	cltest.AssertServerResponse(t, resp, http.StatusOK)
	body := string(cltest.ParseResponseBody(t, resp))
	require.Equal(t, `{"message":"pong"}`, strings.TrimSpace(body))
}

func TestPingController_Show_NoCredentials(t *testing.T) {
	t.Parallel()

	app := cltest.NewApplicationEVMDisabled(t)
	require.NoError(t, app.Start(testutils.Context(t)))

	client := clhttptest.NewTestLocalOnlyHTTPClient()
	url := app.Server.URL + "/v2/ping"
	resp, err := client.Get(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
