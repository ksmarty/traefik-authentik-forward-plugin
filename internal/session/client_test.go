package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/ksmarty/traefik-authentik-forward-plugin/internal/session"
)

func TestNewClient(t *testing.T) {
	t.Run("with no duration", func(t *testing.T) {
		client, err := session.NewClient(context.Background(), 0)

		// check that there is no error
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// check that the client is not nil
		if client == nil {
			t.Fatal("expected client to be not nil")
		}

		// check that the client is a standard client
		if _, ok := client.(*session.StandardClient); !ok {
			t.Fatal("expected client to be a standard client")
		}
	})

	t.Run("with duration", func(t *testing.T) {
		client, err := session.NewClient(context.Background(), 10*time.Second)

		// check that there is no error
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// check that the client is not nil
		if client == nil {
			t.Fatal("expected client to be not nil")
		}

		// check that the client is a cache client
		if _, ok := client.(*session.CacheClient); !ok {
			t.Fatal("expected client to be a cache client")
		}
	})
}
