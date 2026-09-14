package session_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ksmarty/traefik-authentik-forward-plugin/internal/session"
)

func TestNewCacheClient(t *testing.T) {
	t.Run("with no duration", func(t *testing.T) {
		client, err := session.NewCacheClient(context.Background(), 0)

		// check that an error is returned
		if err == nil {
			t.Fatal("expected error, got none")
		}

		// check that the client is nil
		if client != nil {
			t.Fatal("expected client to be nil")
		}
	})

	t.Run("with duration", func(t *testing.T) {
		client, err := session.NewCacheClient(context.Background(), 10*time.Second)

		// check that there is no error
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// check that the client is not nil
		if client == nil {
			t.Fatal("expected client to be not nil")
		}
	})
}

func TestCacheClient(t *testing.T) {
	t.Run("retrieve without store", func(t *testing.T) {
		client, _ := session.NewCacheClient(context.Background(), 10*time.Second)

		session := client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})

		// check that the session is nil
		if session != nil {
			t.Fatal("expected session to be nil")
		}
	})

	t.Run("retrieve after store", func(t *testing.T) {
		client, _ := session.NewCacheClient(context.Background(), 10*time.Second)

		session := &session.Session{
			IsAuthenticated: true,
			Headers: http.Header{
				"X-Test": []string{"test"},
			},
			Cookies: []*http.Cookie{
				{
					Name:  "test",
					Value: "test",
				},
			},
		}
		client.Set([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		}, session)

		// check that the session is not nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session == nil {
			t.Fatal("expected session to be not nil")
		}

		// check that the session has the expected values
		if !session.IsAuthenticated {
			t.Errorf("expected session to be authenticated")
		}

		if session.Headers.Get("X-Test") != "test" {
			t.Errorf("expected session to have original headers")
		}

		if len(session.Cookies) != 1 || session.Cookies[0].Name != "test" || session.Cookies[0].Value != "test" {
			t.Errorf("expected session to have original cookie")
		}
	})

	t.Run("retrieve after delete", func(t *testing.T) {
		client, _ := session.NewCacheClient(context.Background(), 10*time.Second)

		session := &session.Session{
			IsAuthenticated: true,
			Headers:         http.Header{},
			Cookies: []*http.Cookie{
				{
					Name:  "test",
					Value: "test",
				},
			},
		}
		client.Set([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		}, session)
		client.Delete([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})

		// check that the session is nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session != nil {
			t.Errorf("expected session to be nil")
		}
	})

	t.Run("retrieve after expiration", func(t *testing.T) {
		client, _ := session.NewCacheClient(context.Background(), 10*time.Millisecond)

		session := &session.Session{
			IsAuthenticated: true,
			Headers:         http.Header{},
			Cookies:         []*http.Cookie{},
		}
		client.Set([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		}, session)

		// wait for the session to expire
		time.Sleep(30 * time.Millisecond)

		// check that the session is nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session != nil {
			t.Errorf("expected session to be nil")
		}
	})

	t.Run("retrieve after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client, _ := session.NewCacheClient(ctx, 500*time.Millisecond)

		session := &session.Session{
			IsAuthenticated: true,
			Headers:         http.Header{},
			Cookies:         []*http.Cookie{},
		}
		client.Set([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		}, session)

		// cancel the context
		cancel()

		// wait for the cancellation to be processed
		time.Sleep(10 * time.Millisecond)

		// check that the session is not nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session == nil {
			t.Errorf("expected session to be not nil")
		}

		// wait for the session to expire
		time.Sleep(500 * time.Millisecond)

		// check that the session is not nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session == nil {
			t.Errorf("expected session to be not nil")
		}
	})

	t.Run("store after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client, _ := session.NewCacheClient(ctx, 10*time.Second)

		// cancel the context
		cancel()

		session := &session.Session{
			IsAuthenticated: true,
			Headers:         http.Header{},
			Cookies:         []*http.Cookie{},
		}
		client.Set([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		}, session)

		// check that the session is nil
		session = client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session != nil {
			t.Errorf("expected session to be nil")
		}
	})

	t.Run("delete before store", func(t *testing.T) {
		client, _ := session.NewCacheClient(context.Background(), 10*time.Second)

		client.Delete([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})

		// check that the session is nil
		session := client.Get([]*http.Cookie{
			{
				Name:  "test",
				Value: "test",
			},
		})
		if session != nil {
			t.Errorf("expected session to be nil")
		}
	})
}
