package traefik_dynamic_redirects

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type Config struct {
	Redirects []Redirect `json:"redirects,omitempty"`
}

type Redirect struct {
	SourceURL           string            `json:"sourceURL,omitempty"`
	StatusCode          int               `json:"statusCode,omitempty"`
	PreserveQueryString bool              `json:"preserveQueryString,omitempty"`
	SubpathMatching     bool              `json:"subpathMatching,omitempty"`
	AuthenticatedCookie string            `json:"authenticatedCookie,omitempty"`
	AuthenticatedTarget string            `json:"authenticatedTarget,omitempty"`
	LocaleCookie        string            `json:"localeCookie,omitempty"`
	DefaultTarget       string            `json:"defaultTarget,omitempty"`
	LocaleTargets       map[string]string `json:"localeTargets,omitempty"`
}

type Target struct {
	URL                 string
	StatusCode          int
	PreserveQueryString bool
}

type RuntimeRedirect struct {
	StatusCode          int
	PreserveQueryString bool
	AuthenticatedCookie string
	AuthenticatedTarget string
	LocaleCookie        string
	DefaultTarget       string
	LocaleTargets       map[string]string
}

func CreateConfig() *Config {
	return &Config{
		Redirects: []Redirect{},
	}
}

type subpathRedirect struct {
	host     string
	path     string
	redirect RuntimeRedirect
}

type DynamicRedirects struct {
	next             http.Handler
	name             string
	redirects        map[string]RuntimeRedirect
	subpathRedirects []subpathRedirect
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	_ = ctx

	redirects := make(map[string]RuntimeRedirect, len(config.Redirects))
	var subpathRedirects []subpathRedirect

	for _, redirect := range config.Redirects {
		if redirect.SourceURL == "" {
			return nil, fmt.Errorf("sourceURL is required")
		}

		sourceHost, sourcePath, err := parseSourceURL(redirect.SourceURL)
		if err != nil {
			return nil, err
		}

		if err := validateRedirect(redirect.SourceURL, &redirect); err != nil {
			return nil, err
		}

		runtimeRedirect := RuntimeRedirect{
			StatusCode:          redirect.StatusCode,
			PreserveQueryString: redirect.PreserveQueryString,
			AuthenticatedCookie: redirect.AuthenticatedCookie,
			AuthenticatedTarget: redirect.AuthenticatedTarget,
			LocaleCookie:        redirect.LocaleCookie,
			DefaultTarget:       redirect.DefaultTarget,
			LocaleTargets:       normalizeLocaleTargets(redirect.LocaleTargets),
		}

		if redirect.SubpathMatching {
			subpathRedirects = append(subpathRedirects, subpathRedirect{
				host:     sourceHost,
				path:     strings.TrimSuffix(sourcePath, "/"),
				redirect: runtimeRedirect,
			})
			continue
		}

		key := buildKey(sourceHost, sourcePath)
		redirects[key] = runtimeRedirect
	}

	// Longest path first, so the most specific subpath match wins when more
	// than one configured prefix could match the same request.
	sort.Slice(subpathRedirects, func(i, j int) bool {
		return len(subpathRedirects[i].path) > len(subpathRedirects[j].path)
	})

	return &DynamicRedirects{
		next:             next,
		name:             name,
		redirects:        redirects,
		subpathRedirects: subpathRedirects,
	}, nil
}

func (dynamicRedirects *DynamicRedirects) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := normalizeHost(req.Host)
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	key := buildKey(host, path)

	if dynamicRedirect, found := dynamicRedirects.redirects[key]; found {
		if target, found := resolveRedirect(req, dynamicRedirect); found {
			redirect(rw, req, target)
			return
		}
	}

	for _, sr := range dynamicRedirects.subpathRedirects {
		if sr.host != host || !pathMatchesPrefix(path, sr.path) {
			continue
		}

		if target, found := resolveRedirect(req, sr.redirect); found {
			redirect(rw, req, target)
			return
		}

		break
	}

	dynamicRedirects.next.ServeHTTP(rw, req)
}

// pathMatchesPrefix reports whether path is prefixPath itself or a path
// segment nested under it. prefixPath is expected to have no trailing slash
// (the empty string represents the root "/", which matches every path).
// Segment boundaries are enforced so a prefix like "/poll" matches "/poll"
// and "/poll/123" but not "/pollution".
func pathMatchesPrefix(path, prefixPath string) bool {
	if prefixPath == "" {
		return true
	}

	if path == prefixPath {
		return true
	}

	return strings.HasPrefix(path, prefixPath+"/")
}

func redirect(rw http.ResponseWriter, req *http.Request, target Target) {
	targetURL := target.URL

	if target.PreserveQueryString && req.URL.RawQuery != "" {
		separator := "?"
		if strings.Contains(targetURL, "?") {
			separator = "&"
		}

		targetURL += separator + req.URL.RawQuery
	}

	http.Redirect(rw, req, targetURL, target.StatusCode)
}

func resolveRedirect(req *http.Request, dynamicRedirect RuntimeRedirect) (Target, bool) {
	if dynamicRedirect.AuthenticatedCookie != "" && dynamicRedirect.AuthenticatedTarget != "" {
		if cookie, err := req.Cookie(dynamicRedirect.AuthenticatedCookie); err == nil && cookie.Value != "" {
			return Target{
				URL:                 dynamicRedirect.AuthenticatedTarget,
				StatusCode:          dynamicRedirect.StatusCode,
				PreserveQueryString: dynamicRedirect.PreserveQueryString,
			}, true
		}
	}

	if dynamicRedirect.LocaleCookie != "" {
		if cookie, err := req.Cookie(dynamicRedirect.LocaleCookie); err == nil {
			locale := strings.ToLower(cookie.Value)

			if targetURL, found := dynamicRedirect.LocaleTargets[locale]; found {
				return Target{
					URL:                 targetURL,
					StatusCode:          dynamicRedirect.StatusCode,
					PreserveQueryString: dynamicRedirect.PreserveQueryString,
				}, true
			}
		}
	}

	if locale, found := findLocaleFromAcceptLanguage(req.Header.Get("Accept-Language"), dynamicRedirect.LocaleTargets); found {
		return Target{
			URL:                 dynamicRedirect.LocaleTargets[locale],
			StatusCode:          dynamicRedirect.StatusCode,
			PreserveQueryString: dynamicRedirect.PreserveQueryString,
		}, true
	}

	if dynamicRedirect.DefaultTarget != "" {
		return Target{
			URL:                 dynamicRedirect.DefaultTarget,
			StatusCode:          dynamicRedirect.StatusCode,
			PreserveQueryString: dynamicRedirect.PreserveQueryString,
		}, true
	}

	return Target{}, false
}

func parseSourceURL(sourceURL string) (string, string, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid sourceURL %q: %w", sourceURL, err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("sourceURL must be absolute, got %q", sourceURL)
	}

	if parsed.RawQuery != "" {
		return "", "", fmt.Errorf("sourceURL must not contain query string, got %q", sourceURL)
	}

	if parsed.Fragment != "" {
		return "", "", fmt.Errorf("sourceURL must not contain fragment, got %q", sourceURL)
	}

	host := normalizeHost(parsed.Host)
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}

	return host, path, nil
}

func validateRedirect(sourceURL string, redirect *Redirect) error {
	if redirect.StatusCode == 0 {
		redirect.StatusCode = http.StatusFound
	}

	if !isValidRedirectStatusCode(redirect.StatusCode) {
		return fmt.Errorf("invalid statusCode %d for %s", redirect.StatusCode, sourceURL)
	}

	if redirect.DefaultTarget == "" {
		return fmt.Errorf("defaultTarget is required for %s", sourceURL)
	}

	if err := validateTargetURL(redirect.DefaultTarget); err != nil {
		return fmt.Errorf("invalid defaultTarget %q for %s: %w", redirect.DefaultTarget, sourceURL, err)
	}

	if redirect.AuthenticatedTarget != "" {
		if redirect.AuthenticatedCookie == "" {
			return fmt.Errorf("authenticatedCookie is required when authenticatedTarget is set for %s", sourceURL)
		}

		if err := validateTargetURL(redirect.AuthenticatedTarget); err != nil {
			return fmt.Errorf("invalid authenticatedTarget %q for %s: %w", redirect.AuthenticatedTarget, sourceURL, err)
		}
	}

	if redirect.LocaleCookie != "" && len(redirect.LocaleTargets) == 0 {
		return fmt.Errorf("localeTargets is required when localeCookie is set for %s", sourceURL)
	}

	for locale, targetURL := range redirect.LocaleTargets {
		if locale == "" {
			return fmt.Errorf("localeTargets contains empty locale for %s", sourceURL)
		}

		if err := validateTargetURL(targetURL); err != nil {
			return fmt.Errorf("invalid localeTargets[%q]=%q for %s: %w", locale, targetURL, sourceURL, err)
		}
	}

	return nil
}

func validateTargetURL(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("targetURL must be absolute")
	}

	return nil
}

func findLocaleFromAcceptLanguage(header string, supported map[string]string) (string, bool) {
	languages := strings.Split(header, ",")

	for _, language := range languages {
		language = strings.TrimSpace(strings.ToLower(language))
		language = strings.Split(language, ";")[0]

		if language == "" {
			continue
		}

		baseLanguage := strings.Split(language, "-")[0]

		if _, found := supported[baseLanguage]; found {
			return baseLanguage, true
		}
	}

	return "", false
}

func normalizeLocaleTargets(localeTargets map[string]string) map[string]string {
	normalized := make(map[string]string, len(localeTargets))

	for locale, targetURL := range localeTargets {
		normalized[strings.ToLower(locale)] = targetURL
	}

	return normalized
}

func normalizeHost(host string) string {
	host = strings.ToLower(host)

	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}

	return host
}

func buildKey(host, path string) string {
	return host + "\x00" + path
}

func isValidRedirectStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, // 301
		http.StatusFound,             // 302
		http.StatusSeeOther,          // 303
		http.StatusTemporaryRedirect, // 307
		http.StatusPermanentRedirect: // 308
		return true
	default:
		return false
	}
}
