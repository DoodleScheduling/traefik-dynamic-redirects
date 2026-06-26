package traefik_dynamic_redirects

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirectWithAuthenticatedCookie(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/home/")
}

func TestRedirectWithLocaleCookie(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "locale",
		Value: "es",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/es/")
}

func TestRedirectWithLocaleCookieIsCaseInsensitive(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		{
			SourceURL:           "https://example.com/",
			StatusCode:          http.StatusFound,
			PreserveQueryString: "enabled",
			LocaleCookie:        "locale",
			DefaultTarget:       "https://example.com/en/",
			LocaleTargets: map[string]string{
				"DE": "https://example.com/de/",
				"IT": "https://example.com/it/",
				"FR": "https://example.com/fr/",
				"ES": "https://example.com/es/",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "locale",
		Value: "ES",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/es/")
}

func TestRedirectWithAcceptLanguage(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/fr/")
}

func TestRedirectUsesFirstSupportedAcceptLanguage(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		{
			SourceURL:           "https://example.com/",
			StatusCode:          http.StatusFound,
			PreserveQueryString: "enabled",
			DefaultTarget:       "https://example.com/en/",
			LocaleTargets: map[string]string{
				"de": "https://example.com/de/",
				"fr": "https://example.com/fr/",
				"es": "https://example.com/es/",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.Header.Set("Accept-Language", "nl-NL,nl;q=0.9,es-ES;q=0.8,fr-FR;q=0.7")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/es/")
}

func TestRedirectFallsBackToDefaultTarget(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.Header.Set("Accept-Language", "nl-NL,nl;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/")
}

func TestAuthenticatedCookieHasPriorityOverLocaleCookie(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	req.AddCookie(&http.Cookie{
		Name:  "locale",
		Value: "es",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/home/")
}

func TestLocaleCookieHasPriorityOverAcceptLanguage(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "locale",
		Value: "es",
	})
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/es/")
}

func TestRedirectPreservesQueryString(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?utm_source=google&campaign=test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/home/?utm_source=google&campaign=test")
}

func TestRedirectDoesNotPreserveQueryStringWhenDisabled(t *testing.T) {
	redirect := testRedirect()
	redirect.PreserveQueryString = "disabled"

	handler := newTestHandler(t, []Redirect{
		redirect,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?utm_source=google", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/home/")
}

func TestRedirectAppendsQueryStringWithAmpersandWhenTargetAlreadyHasQueryString(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		{
			SourceURL:           "https://example.com/",
			StatusCode:          http.StatusFound,
			PreserveQueryString: "enabled",
			AuthenticatedCookie: "Authentication",
			AuthenticatedTarget: "https://example.com/home/?source=plugin",
			DefaultTarget:       "https://example.com/en/",
			LocaleTargets:       map[string]string{},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?utm_source=google", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/home/?source=plugin&utm_source=google")
}

func TestRedirectOnlyMatchesConfiguredSourcePath(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/random", nil)
	req.AddCookie(&http.Cookie{
		Name:  "Authentication",
		Value: "abc",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusTeapot)
}

func TestPassThroughWhenRedirectIsNotFound(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.org/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusTeapot)
}

func TestRequestHostIsNormalizedWhenItContainsPort(t *testing.T) {
	handler := newTestHandler(t, []Redirect{
		testRedirect(),
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com:443/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/")
}

func TestSourceURLHostIsNormalizedWhenConfiguredHostHasUppercase(t *testing.T) {
	redirect := testRedirect()
	redirect.SourceURL = "https://EXAMPLE.com/"

	handler := newTestHandler(t, []Redirect{
		redirect,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/")
}

func TestDefaultStatusCodeIsFound(t *testing.T) {
	redirect := testRedirect()
	redirect.StatusCode = 0

	handler := newTestHandler(t, []Redirect{
		redirect,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/")
}

func TestDefaultPreserveQueryStringIsEnabled(t *testing.T) {
	redirect := testRedirect()
	redirect.PreserveQueryString = ""

	handler := newTestHandler(t, []Redirect{
		redirect,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?utm_source=google", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/?utm_source=google")
}

func TestPreserveQueryStringIsCaseInsensitive(t *testing.T) {
	redirect := testRedirect()
	redirect.PreserveQueryString = "ENABLED"

	handler := newTestHandler(t, []Redirect{
		redirect,
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/?utm_source=google", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusFound)
	assertLocation(t, rec, "https://example.com/en/?utm_source=google")
}

func TestNewReturnsErrorForInvalidStatusCode(t *testing.T) {
	redirect := testRedirect()
	redirect.StatusCode = http.StatusOK

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "invalid statusCode")
}

func TestNewReturnsErrorWhenSourceURLIsMissing(t *testing.T) {
	redirect := testRedirect()
	redirect.SourceURL = ""

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "sourceURL is required")
}

func TestNewReturnsErrorWhenSourceURLIsNotAbsolute(t *testing.T) {
	redirect := testRedirect()
	redirect.SourceURL = "example.com"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "sourceURL must be absolute")
}

func TestNewReturnsErrorWhenSourceURLContainsQueryString(t *testing.T) {
	redirect := testRedirect()
	redirect.SourceURL = "https://example.com/?utm=test"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "sourceURL must not contain query string")
}

func TestNewReturnsErrorWhenSourceURLContainsFragment(t *testing.T) {
	redirect := testRedirect()
	redirect.SourceURL = "https://example.com/#section"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "sourceURL must not contain fragment")
}

func TestNewReturnsErrorWhenDefaultTargetIsMissing(t *testing.T) {
	redirect := testRedirect()
	redirect.DefaultTarget = ""

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "defaultTarget is required")
}

func TestNewReturnsErrorWhenDefaultTargetIsNotAbsolute(t *testing.T) {
	redirect := testRedirect()
	redirect.DefaultTarget = "/en/"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "invalid defaultTarget")
}

func TestNewReturnsErrorForInvalidPreserveQueryString(t *testing.T) {
	redirect := testRedirect()
	redirect.PreserveQueryString = "true"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "invalid preserveQueryString")
}

func TestNewReturnsErrorWhenAuthenticatedTargetHasNoCookie(t *testing.T) {
	redirect := testRedirect()
	redirect.AuthenticatedCookie = ""

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "authenticatedCookie is required")
}

func TestNewReturnsErrorWhenAuthenticatedTargetIsNotAbsolute(t *testing.T) {
	redirect := testRedirect()
	redirect.AuthenticatedTarget = "/home/"

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "invalid authenticatedTarget")
}

func TestNewReturnsErrorWhenLocaleCookieHasNoLocaleTargets(t *testing.T) {
	redirect := testRedirect()
	redirect.LocaleTargets = nil

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "localeTargets is required")
}

func TestNewReturnsErrorWhenLocaleTargetIsNotAbsolute(t *testing.T) {
	redirect := testRedirect()
	redirect.LocaleTargets = map[string]string{
		"es": "/es/",
	}

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "invalid localeTargets")
}

func TestNewReturnsErrorWhenLocaleTargetHasEmptyLocale(t *testing.T) {
	redirect := testRedirect()
	redirect.LocaleTargets = map[string]string{
		"": "https://example.com/es/",
	}

	_, err := New(context.Background(), nextHandler(), &Config{
		Redirects: []Redirect{
			redirect,
		},
	}, "dynamic-redirects")

	assertErrorContains(t, err, "localeTargets contains empty locale")
}

func TestParseSourceURLDefaultsEmptyPathToRoot(t *testing.T) {
	host, path, err := parseSourceURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	if host != "example.com" {
		t.Fatalf("expected host %q, got %q", "example.com", host)
	}

	if path != "/" {
		t.Fatalf("expected path %q, got %q", "/", path)
	}
}

func TestIsValidRedirectStatusCode(t *testing.T) {
	validCodes := []int{
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
	}

	for _, code := range validCodes {
		if !isValidRedirectStatusCode(code) {
			t.Fatalf("expected status code %d to be valid", code)
		}
	}

	invalidCodes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusInternalServerError,
	}

	for _, code := range invalidCodes {
		if isValidRedirectStatusCode(code) {
			t.Fatalf("expected status code %d to be invalid", code)
		}
	}
}

func newTestHandler(t *testing.T, redirects []Redirect) http.Handler {
	t.Helper()

	handler, err := New(context.Background(), nextHandler(), &Config{
		Redirects: redirects,
	}, "dynamic-redirects")
	if err != nil {
		t.Fatal(err)
	}

	return handler
}

func nextHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	})
}

func testRedirect() Redirect {
	return Redirect{
		SourceURL:           "https://example.com/",
		StatusCode:          http.StatusFound,
		PreserveQueryString: "enabled",
		AuthenticatedCookie: "Authentication",
		AuthenticatedTarget: "https://example.com/home/",
		LocaleCookie:        "locale",
		DefaultTarget:       "https://example.com/en/",
		LocaleTargets: map[string]string{
			"de": "https://example.com/de/",
			"it": "https://example.com/it/",
			"fr": "https://example.com/fr/",
			"es": "https://example.com/es/",
		},
	}
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()

	if rec.Code != expected {
		t.Fatalf("expected status %d, got %d", expected, rec.Code)
	}
}

func assertLocation(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()

	if got := rec.Header().Get("Location"); got != expected {
		t.Fatalf("expected Location %q, got %q", expected, got)
	}
}

func assertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error to contain %q, got %q", expected, err.Error())
	}
}
