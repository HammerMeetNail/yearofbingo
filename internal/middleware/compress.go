package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// gzipResponseWriter delays the final header until the first body bytes so
// content sniffing sees the original representation, not a gzip header.
type gzipResponseWriter struct {
	http.ResponseWriter
	eligible  bool
	status    int
	headers   http.Header
	committed bool
	writer    *gzip.Writer
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }

func (g *gzipResponseWriter) WriteHeader(status int) {
	if g.status != 0 {
		return
	}
	if status < 100 || status > 999 {
		panic("invalid WriteHeader code")
	}
	if status == http.StatusSwitchingProtocols {
		g.status = status
		g.committed = true
		g.ResponseWriter.WriteHeader(status)
		return
	}
	if status < 200 {
		g.ResponseWriter.WriteHeader(status)
		return
	}
	g.status = status
	if !g.eligible {
		addAcceptEncodingVary(g.Header())
		g.committed = true
		g.ResponseWriter.WriteHeader(status)
		return
	}
	g.headers = g.Header().Clone()
}

func (g *gzipResponseWriter) commit(body []byte) {
	if g.committed {
		return
	}
	if g.status == 0 {
		g.WriteHeader(http.StatusOK)
	}
	if g.committed {
		return
	}
	h := g.Header()
	clear(h)
	for key, values := range g.headers {
		h[key] = values
	}
	addAcceptEncodingVary(h)
	if g.eligible && len(body) > 0 && g.status != http.StatusNoContent &&
		g.status != http.StatusNotModified && g.status != http.StatusPartialContent &&
		h.Get("Content-Encoding") == "" && h.Get("Content-Range") == "" {
		if _, exists := h["Content-Type"]; !exists {
			h.Set("Content-Type", http.DetectContentType(body))
		}
		if !isPreCompressedType(h.Get("Content-Type")) {
			h.Del("Content-Length")
			h.Set("Content-Encoding", "gzip")
			g.writer = gzipPool.Get().(*gzip.Writer)
			g.writer.Reset(g.ResponseWriter)
		}
	}
	g.committed = true
	g.ResponseWriter.WriteHeader(g.status)
}

func (g *gzipResponseWriter) Write(body []byte) (int, error) {
	g.commit(body)
	if g.writer != nil {
		return g.writer.Write(body)
	}
	return g.ResponseWriter.Write(body)
}

func (g *gzipResponseWriter) Flush() { _ = g.FlushError() }

func (g *gzipResponseWriter) FlushError() error {
	g.commit(nil)
	if g.writer != nil {
		if err := g.writer.Flush(); err != nil {
			return err
		}
	}
	return http.NewResponseController(g.ResponseWriter).Flush()
}

// Pool writers only while a response actually needs compression.
var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

type Compress struct{}

func NewCompress() *Compress { return &Compress{} }

func (c *Compress) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g := &gzipResponseWriter{
			ResponseWriter: w,
			eligible: r.Method != http.MethodHead && r.Header.Get("Range") == "" &&
				acceptsGzip(strings.Join(r.Header.Values("Accept-Encoding"), ",")) && !isPreCompressedPath(r.URL.Path),
		}
		defer func() {
			if g.writer != nil {
				_ = g.writer.Close()
				g.writer.Reset(io.Discard)
				gzipPool.Put(g.writer)
			}
		}()
		next.ServeHTTP(g, r)
		g.commit(nil)
	})
}

func addAcceptEncodingVary(h http.Header) {
	for _, value := range h.Values("Vary") {
		for _, token := range strings.Split(value, ",") {
			if token = strings.TrimSpace(token); token == "*" || strings.EqualFold(token, "Accept-Encoding") {
				return
			}
		}
	}
	h.Add("Vary", "Accept-Encoding")
}

func acceptsGzip(value string) bool {
	explicit, wildcard := -1.0, 0.0
	for _, item := range strings.Split(value, ",") {
		parts := strings.Split(item, ";")
		token := strings.TrimSpace(parts[0])
		if !strings.EqualFold(token, "gzip") && token != "*" {
			continue
		}
		quality := 1.0
		for _, parameter := range parts[1:] {
			key, raw, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if strings.EqualFold(strings.TrimSpace(key), "q") {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
				if !ok || err != nil || !(parsed >= 0 && parsed <= 1) {
					quality = 0
				} else {
					quality = parsed
				}
			}
		}
		if strings.EqualFold(token, "gzip") {
			explicit = quality
		} else {
			wildcard = quality
		}
	}
	if explicit >= 0 {
		return explicit > 0
	}
	return wildcard > 0
}

func isPreCompressedType(value string) bool {
	mediaType, _, _ := strings.Cut(strings.ToLower(value), ";")
	mediaType = strings.TrimSpace(mediaType)
	if strings.HasPrefix(mediaType, "image/") {
		return mediaType != "image/svg+xml"
	}
	if strings.HasPrefix(mediaType, "audio/") || strings.HasPrefix(mediaType, "video/") {
		return true
	}
	switch mediaType {
	case "application/zip", "application/gzip", "application/x-gzip", "application/x-7z-compressed", "application/x-rar-compressed", "application/zstd", "font/woff", "font/woff2":
		return true
	}
	return false
}

// isPreCompressedPath returns true for file types that are already compressed.
func isPreCompressedPath(path string) bool {
	compressedExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico",
		".mp4", ".webm", ".mp3", ".ogg",
		".zip", ".gz", ".br", ".zst",
		".woff", ".woff2",
	}

	lowerPath := strings.ToLower(path)
	for _, ext := range compressedExtensions {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}
	return false
}
