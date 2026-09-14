package httputil_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksmarty/traefik-authentik-forward-plugin/internal/httputil"
)

func TestResponseMangler(t *testing.T) {
	t.Run("with mangle function", func(t *testing.T) {
		mangleCalled := false
		mangleFunc := func(rw http.ResponseWriter) {
			mangleCalled = true

			rw.Header().Set("X-Mangle-Test", "test_value")
		}

		responseWriter := httptest.NewRecorder()
		responseMangler := &httputil.ResponseMangler{
			ResponseWriter: responseWriter,
			MangleFunc:     mangleFunc,
		}

		responseMangler.WriteHeader(http.StatusOK)

		// check that the response code is the received one
		expectedCode := http.StatusOK
		if responseWriter.Code != expectedCode {
			t.Errorf("expected status code %d, got %d", expectedCode, responseWriter.Code)
		}

		// check that the mangle function was called
		if !mangleCalled {
			t.Fatalf("expected mangle function to be called")
		}

		// check that the response mangling is applied
		if responseWriter.Header().Get("X-Mangle-Test") != "test_value" {
			t.Errorf("expected X-Mangle-Test header to be set, got %s", responseWriter.Header().Get("X-Mangle-Test"))
		}
	})
}

func TestResponseManglerUnwrap(t *testing.T) {
	responseWriter := httptest.NewRecorder()
	responseMangler := &httputil.ResponseMangler{
		ResponseWriter: responseWriter,
		MangleFunc:     func(rw http.ResponseWriter) {},
	}

	// Unwrap must expose the wrapped writer so http.ResponseController can
	// reach it (used by Traefik to upgrade connections).
	if unwrapped := responseMangler.Unwrap(); unwrapped != http.ResponseWriter(responseWriter) {
		t.Fatalf("Unwrap must return the wrapped ResponseWriter")
	}
}

func TestResponseManglerFlush(t *testing.T) {
	responseWriter := httptest.NewRecorder()
	responseMangler := &httputil.ResponseMangler{
		ResponseWriter: responseWriter,
		MangleFunc:     func(rw http.ResponseWriter) {},
	}

	responseMangler.Flush()

	if !responseWriter.Flushed {
		t.Fatalf("Flush must be forwarded to the wrapped ResponseWriter")
	}
}

func TestResponseManglerHijackUnsupported(t *testing.T) {
	responseMangler := &httputil.ResponseMangler{
		ResponseWriter: httptest.NewRecorder(),
		MangleFunc:     func(rw http.ResponseWriter) {},
	}

	if _, _, err := responseMangler.Hijack(); err == nil {
		t.Fatalf("Hijack must fail when the wrapped ResponseWriter is not hijackable")
	}
}

// fakeResponseWriter is a ResponseWriter that supports hijacking and flushing.
// It is used to check that ResponseMangler forwards those calls.
type fakeResponseWriter struct {
	header   http.Header
	hijacked bool
	flushed  bool
}

func (f *fakeResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = http.Header{}
	}

	return f.header
}

func (f *fakeResponseWriter) Write(b []byte) (int, error) { return len(b), nil }

func (f *fakeResponseWriter) WriteHeader(int) {}

func (f *fakeResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true

	return nil, nil, nil
}

func (f *fakeResponseWriter) Flush() { f.flushed = true }

func TestResponseManglerHijackDelegates(t *testing.T) {
	responseWriter := &fakeResponseWriter{}
	responseMangler := &httputil.ResponseMangler{
		ResponseWriter: responseWriter,
		MangleFunc:     func(rw http.ResponseWriter) {},
	}

	if _, _, err := responseMangler.Hijack(); err != nil {
		t.Fatalf("Hijack returned unexpected error: %v", err)
	}
	if !responseWriter.hijacked {
		t.Fatalf("Hijack must be forwarded to the wrapped ResponseWriter")
	}
}
