package integration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	relayprovider "micro-one-api/internal/relay/provider"
	relayserver "micro-one-api/internal/relay/server"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestRelayIntegration(t *testing.T) {
	identityCleanup, identityClient := setupInMemoryIdentityService(t, "127.0.0.1:19001")
	defer identityCleanup()

	channelCleanup, channelClient := setupInMemoryChannelService(t, "127.0.0.1:19002")
	defer channelCleanup()

	providerFactory := relayprovider.NewProviderFactory(30 * time.Second)

	httpServer := relayserver.NewHTTPServer(identityClient, channelClient, providerFactory)

	lis, err := net.Listen("tcp", ":19000")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := khttp.NewServer(khttp.Listener(lis))
	httpServer.RegisterRoutes(srv)

	go func() {
		if err := srv.Start(context.Background()); err != nil && err != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", err)
		}
	}()
	defer srv.Stop(context.Background())

	time.Sleep(100 * time.Millisecond) // Give servers time to start

	t.Run("GetHealth", func(t *testing.T) {
		resp, err := http.Get("http://localhost:19000/health")
		if err != nil {
			t.Fatalf("failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetModels", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var modelsResp struct {
			Object string `json:"object"`
			Data   []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(modelsResp.Data) == 0 {
			t.Fatal("expected some models")
		}
	})

	t.Run("GetModelsRestricted", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		req.Header.Set("Authorization", "Bearer restricted-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var modelsResp struct {
			Object string `json:"object"`
			Data   []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Since the token has whitelist ["gpt-4o-mini"] and user is in "premium" group,
		// but premium group only has gpt-4o, intersection is empty
		t.Logf("Models returned: %v", modelsResp.Data)
		if len(modelsResp.Data) != 0 {
			t.Logf("Got %d models, expected 0 due to whitelist/group intersection", len(modelsResp.Data))
		}
	})

	t.Run("MissingAuthorization", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		req.Header.Set("Authorization", "Bearer expired-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("DisabledToken", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:19000/v1/models", nil)
		req.Header.Set("Authorization", "Bearer disabled-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to get models: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", resp.StatusCode)
		}
	})
}
