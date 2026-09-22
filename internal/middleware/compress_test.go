package middleware

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCompress_GzipWhenAccepted(t *testing.T) {
	compress := NewCompress()

	responseBody := "This is a test response that should be compressed"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	compress.Apply(handler).ServeHTTP(rr, req)

	// Check Content-Encoding header
	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %q", got)
	}

	// Check Vary header
	if got := rr.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("expected Vary: Accept-Encoding, got %q", got)
	}

	// Decompress and verify content
	gzReader, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzReader.Close() }()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	if string(decompressed) != responseBody {
		t.Errorf("expected %q, got %q", responseBody, string(decompressed))
	}
}

func TestCompress_NoGzipWhenNotAccepted(t *testing.T) {
	compress := NewCompress()

	responseBody := "This is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// No Accept-Encoding header

	rr := httptest.NewRecorder()
	compress.Apply(handler).ServeHTTP(rr, req)

	// Check no Content-Encoding header
	if got := rr.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("expected no Content-Encoding, got %q", got)
	}

	// Body should be uncompressed
	if got := rr.Body.String(); got != responseBody {
		t.Errorf("expected %q, got %q", responseBody, got)
	}
}

func TestCompress_SkipPreCompressedFiles(t *testing.T) {
	compress := NewCompress()

	preCompressedPaths := []string{
		"/static/image.jpg",
		"/static/image.jpeg",
		"/static/image.png",
		"/static/image.gif",
		"/static/image.webp",
		"/static/icon.ico",
		"/static/video.mp4",
		"/static/video.webm",
		"/static/audio.mp3",
		"/static/audio.ogg",
		"/static/archive.zip",
		"/static/archive.gz",
		"/static/archive.br",
		"/static/archive.zst",
		"/static/font.woff",
		"/static/font.woff2",
	}

	for _, path := range preCompressedPaths {
		t.Run(path, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("content"))
			})

			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rr := httptest.NewRecorder()
			compress.Apply(handler).ServeHTTP(rr, req)

			// Should not be compressed
			if got := rr.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("path %s should not be compressed, got Content-Encoding: %q", path, got)
			}
		})
	}
}

func TestCompress_CompressTextPaths(t *testing.T) {
	compress := NewCompress()

	textPaths := []string{
		"/api/test",
		"/static/script.js",
		"/static/style.css",
		"/",
		"/about",
	}

	for _, path := range textPaths {
		t.Run(path, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("content"))
			})

			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rr := httptest.NewRecorder()
			compress.Apply(handler).ServeHTTP(rr, req)

			// Should be compressed
			if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
				t.Errorf("path %s should be compressed, got Content-Encoding: %q", path, got)
			}
		})
	}
}

func TestCompress_CaseInsensitiveExtension(t *testing.T) {
	compress := NewCompress()

	// Test uppercase extensions
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	})

	uppercasePaths := []string{
		"/static/image.JPG",
		"/static/image.PNG",
		"/static/font.WOFF2",
	}

	for _, path := range uppercasePaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rr := httptest.NewRecorder()
			compress.Apply(handler).ServeHTTP(rr, req)

			// Should not be compressed (case insensitive check)
			if got := rr.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("path %s should not be compressed, got Content-Encoding: %q", path, got)
			}
		})
	}
}

func TestIsPreCompressedPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/static/image.jpg", true},
		{"/static/image.JPEG", true},
		{"/static/font.woff", true},
		{"/static/font.woff2", true},
		{"/static/script.js", false},
		{"/static/style.css", false},
		{"/api/users", false},
		{"/", false},
		{"/static/image.jpgx", false}, // Not a real extension
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isPreCompressedPath(tt.path)
			if got != tt.expected {
				t.Errorf("isPreCompressedPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestCompress_GzipDeflateAccepted(t *testing.T) {
	compress := NewCompress()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	rr := httptest.NewRecorder()
	compress.Apply(handler).ServeHTTP(rr, req)

	// Should still use gzip
	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %q", got)
	}
}

func TestCompress_VaryHeader(t *testing.T) {
	compress := NewCompress()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test content"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	compress.Apply(handler).ServeHTTP(rr, req)

	// Vary header should be set
	if got := rr.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("expected Vary: Accept-Encoding, got %q", got)
	}
}

func TestCompress_Negotiation(t *testing.T) {
	for _, tc := range []struct {
		value string
		gzip  bool
	}{
		{"", false}, {"gzip;q=0", false}, {"xgzip", false}, {"gzip-extra", false},
		{"br, GZIP; q=0.5", true}, {"*;q=0.8", true}, {"*;q=0", false},
		{"gzip;q=0, *;q=1", false}, {"gzip;q=bogus", false}, {"gzip;q=NaN", false},
		{"gzip;q=2", false}, {"gzip;q=1, *;q=0", true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tc.value)
			rr := httptest.NewRecorder()
			NewCompress().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Vary", "Origin")
				_, _ = io.WriteString(w, "hello")
			})).ServeHTTP(rr, req)
			if got := rr.Result().Header.Get("Content-Encoding") == "gzip"; got != tc.gzip {
				t.Fatalf("gzip = %v, want %v", got, tc.gzip)
			}
			if got := strings.Join(rr.Result().Header.Values("Vary"), ", "); got != "Origin, Accept-Encoding" {
				t.Fatalf("Vary = %q", got)
			}
		})
	}
}

func TestCompress_ServeContentRoundTrip(t *testing.T) {
	body := strings.Repeat("a moderately compressible stylesheet body;", 1000)
	server := httptest.NewServer(NewCompress().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		http.ServeContent(w, r, "style.css", time.Time{}, strings.NewReader(body))
	})))
	defer server.Close()
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	defer client.CloseIdleConnections()
	for _, ranged := range []bool{false, true} {
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept-Encoding", "gzip")
		expected := body
		if ranged {
			req.Header.Set("Range", "bytes=0-9")
			expected = body[:10]
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var reader io.Reader = resp.Body
		if !ranged {
			if resp.Header.Get("Content-Encoding") != "gzip" {
				t.Fatal("missing gzip encoding")
			}
			if resp.Header.Get("Content-Length") == strconv.Itoa(len(body)) {
				t.Fatal("retained original content length")
			}
			gz, err := gzip.NewReader(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gz.Close()
			reader = gz
		} else if resp.StatusCode != http.StatusPartialContent || resp.Header.Get("Content-Encoding") != "" {
			t.Fatalf("invalid range response: %d, %v", resp.StatusCode, resp.Header)
		}
		decoded, err := io.ReadAll(reader)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(decoded) != expected {
			t.Fatalf("roundtrip body length = %d, want %d", len(decoded), len(expected))
		}
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/css") {
			t.Fatalf("Content-Type = %q", resp.Header.Get("Content-Type"))
		}
	}
}

func TestCompress_ResponseExclusions(t *testing.T) {
	for _, tc := range []struct {
		name, method, contentType, encoding string
		status                              int
		write                               bool
	}{
		{name: "head", method: http.MethodHead, status: 200},
		{name: "no content", status: 204}, {name: "not modified", status: 304},
		{name: "empty", status: 200}, {name: "encoded", status: 200, encoding: "br", write: true},
		{name: "image", status: 200, contentType: "image/png", write: true},
		{name: "partial", status: 206, write: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()
			NewCompress().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				if tc.encoding != "" {
					w.Header().Set("Content-Encoding", tc.encoding)
				}
				w.WriteHeader(tc.status)
				if tc.write {
					_, _ = io.WriteString(w, "original body")
				}
			})).ServeHTTP(rr, req)
			if rr.Code != tc.status {
				t.Fatalf("status = %d", rr.Code)
			}
			if got := rr.Result().Header.Get("Content-Encoding"); got != tc.encoding {
				t.Fatalf("Content-Encoding = %q", got)
			}
			expected := ""
			if tc.write {
				expected = "original body"
			}
			if rr.Body.String() != expected {
				t.Fatalf("body = %q", rr.Body.String())
			}
		})
	}
}

func TestCompress_ExplicitHeadersAndSniffing(t *testing.T) {
	body := "<html><body>Hello</body></html>"
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	NewCompress().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Vary", "Origin, accept-encoding")
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("X-Too-Late", "ignored")
		_, _ = io.WriteString(w, body)
	})).ServeHTTP(rr, req)
	h := rr.Result().Header
	if rr.Code != http.StatusCreated || h.Get("Content-Length") != "" || h.Get("X-Too-Late") != "" {
		t.Fatalf("unexpected response: %d %v", rr.Code, h)
	}
	if got := h.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := h.Values("Vary"); len(got) != 1 || got[0] != "Origin, accept-encoding" {
		t.Fatalf("Vary = %v", got)
	}
	gz, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	decoded, err := io.ReadAll(gz)
	if err != nil || string(decoded) != body {
		t.Fatalf("decoded = %q, error = %v", decoded, err)
	}
}

func TestCompress_FlushThroughRequestLogger(t *testing.T) {
	for _, encoding := range []string{"gzip", "identity"} {
		t.Run(encoding, func(t *testing.T) {
			release := make(chan struct{})
			flushed := make(chan error, 1)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := io.WriteString(w, "first chunk\n"); err != nil {
					flushed <- err
					return
				}
				flushed <- http.NewResponseController(w).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
				}
			})
			server := httptest.NewServer(NewRequestLogger(nil).Apply(NewCompress().Apply(handler)))
			defer server.Close()
			// Release before server.Close even when an assertion fails; the request
			// context also bounds both header receipt and the streaming body read.
			defer close(release)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Accept-Encoding", encoding)
			transport := &http.Transport{DisableCompression: true}
			defer transport.CloseIdleConnections()
			resp, err := (&http.Client{Transport: transport}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			select {
			case err := <-flushed:
				if err != nil {
					t.Fatalf("Flush: %v", err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var reader io.Reader = resp.Body
			if encoding == "gzip" {
				gz, err := gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				defer gz.Close()
				reader = gz
			}
			chunk := make([]byte, len("first chunk\n"))
			if _, err := io.ReadFull(reader, chunk); err != nil {
				t.Fatalf("read before handler release: %v", err)
			}
			if string(chunk) != "first chunk\n" {
				t.Fatalf("chunk = %q", chunk)
			}
		})
	}
}
