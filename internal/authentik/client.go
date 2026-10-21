package authentik

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ksmarty/traefik-authentik-forward-plugin/internal/session"
)

type Client struct {
	config  *Config
	client  *http.Client
	session session.Client
}

func NewClient(context context.Context, client *http.Client, config *Config) (*Client, error) {
	sessionClient, err := session.NewClient(context, config.CacheDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create session client: %w", err)
	}

	return &Client{
		config:  config,
		client:  client,
		session: sessionClient,
	}, nil
}

func (c *Client) Check(meta *RequestMeta) (*ResponseMeta, error) {
	// check if s is already cached
	if s := c.session.Get(meta.Cookies); s != nil {
		return &ResponseMeta{
			URL:     meta.URL,
			Cached:  true,
			Session: s,
		}, nil
	}

	// send request to authentik to check if request is authenticated
	res, err := c.request(meta, NginxPath, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	var s *session.Session
	switch res.StatusCode {
	case http.StatusUnauthorized:
		s = &session.Session{
			IsAuthenticated: false,
			Headers:         nil,
			Cookies:         GetCookies(res),
		}
	case http.StatusOK:
		s = &session.Session{
			IsAuthenticated: true,
			Headers:         GetHeaders(res),
			Cookies:         GetCookies(res),
		}
	default:
		return nil, unexpectedStatus(res, NginxPath)
	}

	// cache session
	c.session.Set(meta.Cookies, s)

	return &ResponseMeta{
		URL:     meta.URL,
		Cached:  false,
		Session: s,
	}, nil
}

// maxErrorBodySize caps how much of authentik's response body an error quotes,
// so a large HTML error page cannot flood the caller.
const maxErrorBodySize = 256

// unexpectedStatus reports an authentik response that is neither 200 (allowed)
// nor 401 (challenged).
//
// The body is quoted because it is the only clue to why authentik refused: a
// plain-text "404 page not found" means authentik's own router answered, which
// happens when the traefik outpost is missing for the host, or when authentik's
// core is down, and not when the upstream is unreachable. Without the body all
// of those collapse into the bare "unexpected response: 404".
func unexpectedStatus(res *http.Response, path string) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxErrorBodySize))

	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = http.StatusText(res.StatusCode)
	}

	return fmt.Errorf("unexpected response from authentik for %s: %d %s", path, res.StatusCode, detail)
}

func (c *Client) Request(meta *RequestMeta, path string, query string) (*http.Response, error) {
	// delete session if already cached
	c.session.Delete(meta.Cookies)

	return c.request(meta, path, query)
}

func (c *Client) request(meta *RequestMeta, path string, query string) (*http.Response, error) {
	// send request to authentik
	akReq, err := http.NewRequest(http.MethodGet, c.config.Address+path, nil)
	if err != nil {
		return nil, err
	}

	akReq.URL.RawQuery = query

	// add downstream request metadata
	akReq.Header.Set("X-Forwarded-Host", meta.URL.Host)
	akReq.Header.Set("X-Original-Url", meta.URL.String())

	// add downstream authentik session cookies
	for _, c := range meta.Cookies {
		akReq.AddCookie(c)
	}

	res, err := c.client.Do(akReq)
	if err != nil {
		return nil, err
	}

	if err := c.mangleLocation(meta, res); err != nil {
		return nil, err
	}

	c.mangleCookies(meta, res)

	return res, nil
}

func (c *Client) mangleLocation(meta *RequestMeta, res *http.Response) error {
	loc := res.Header.Get("Location")
	if loc == "" {
		res.Header.Del("Location")
		return nil
	}

	if strings.HasPrefix(loc, c.config.Address+BasePath) {
		// convert absolute outpost redirects to downstream host
		loc = strings.TrimPrefix(loc, c.config.Address)

		locURL, err := url.Parse(loc)
		if err != nil {
			return err
		}

		locURL.Scheme = meta.URL.Scheme
		locURL.Host = meta.URL.Host

		loc = locURL.String()
	}

	res.Header.Set("Location", loc)

	return nil
}

func (c *Client) mangleCookies(meta *RequestMeta, res *http.Response) {
	// get authentik session cookies from response
	cookies := GetCookies(res)

	// set cookie attributes
	for _, cookie := range cookies {
		cookie.HttpOnly = true
		cookie.Secure = meta.URL.Scheme == "https"
	}

	res.Header.Del("Set-Cookie")
	for _, cookie := range cookies {
		res.Header.Add("Set-Cookie", cookie.String())
	}
}
