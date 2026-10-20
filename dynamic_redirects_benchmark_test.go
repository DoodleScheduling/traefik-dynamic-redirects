package traefik_dynamic_redirects

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// BenchmarkSubpathMatchFound measures ServeHTTP for a request that resolves
// via the deepest configured subpath redirect, as the number of unrelated
// subpathMatching entries grows. findPrefixRedirect looks up candidates by
// map key while walking up the request path's own segments, so its cost
// tracks the path's depth, not the number of configured rules -- this should
// stay essentially flat across rule counts, unlike a linear scan over every
// subpathMatching entry would.
func BenchmarkSubpathMatchFound(b *testing.B) {
	for _, count := range []int{10, 100, 1000, 10000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			handler := newBenchmarkHandler(b, count)
			req := httptest.NewRequest(http.MethodGet, "https://benchmark.example/poll/abc123xyz", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

// BenchmarkSubpathMatchNotFound is the worst case for findPrefixRedirect: the
// request path never matches any configured prefix, so it walks all the way
// up to "/" before falling through to the next handler.
func BenchmarkSubpathMatchNotFound(b *testing.B) {
	for _, count := range []int{10, 100, 1000, 10000} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			handler := newBenchmarkHandler(b, count)
			req := httptest.NewRequest(http.MethodGet, "https://benchmark.example/unrelated/nested/path/abc123xyz", nil)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		})
	}
}

func newBenchmarkHandler(b *testing.B, subpathRuleCount int) http.Handler {
	b.Helper()

	redirects := make([]Redirect, 0, subpathRuleCount+1)
	for i := 0; i < subpathRuleCount; i++ {
		redirects = append(redirects, Redirect{
			SourceURL:       "https://benchmark.example/rule-" + strconv.Itoa(i),
			StatusCode:      http.StatusMovedPermanently,
			SubpathMatching: true,
			DefaultTarget:   "https://benchmark.example/target-" + strconv.Itoa(i) + "/",
		})
	}

	// The entry the "found" benchmark is expected to resolve through, mixed
	// in among the unrelated rules rather than appended last.
	redirects = append(redirects, Redirect{
		SourceURL:       "https://benchmark.example/poll",
		StatusCode:      http.StatusMovedPermanently,
		SubpathMatching: true,
		DefaultTarget:   "https://benchmark.example/en/poll-archived/",
	})

	handler, err := New(context.Background(), nextHandler(), &Config{Redirects: redirects}, "dynamic-redirects")
	if err != nil {
		b.Fatal(err)
	}

	return handler
}
