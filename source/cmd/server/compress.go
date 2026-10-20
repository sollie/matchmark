package main

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

const minCompressSize = 256

var compressibleTypes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"application/xml",
	"application/manifest+json",
	"application/wasm",
	"image/svg+xml",
}

func compression(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !enabled {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead || r.Header.Get("Range") != "" || r.Header.Get("Upgrade") != "" {
				next.ServeHTTP(w, r)
				return
			}
			enc := negotiateEncoding(r.Header.Get("Accept-Encoding"))
			cw := &compressWriter{ResponseWriter: w, enc: enc}
			defer cw.close()
			next.ServeHTTP(cw, r)
		})
	}
}

func negotiateEncoding(header string) string {
	if header == "" {
		return ""
	}
	q := map[string]float64{}
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		name = strings.ToLower(strings.TrimSpace(name))
		val := 1.0
		if k, v, ok := strings.Cut(strings.TrimSpace(params), "="); ok && strings.EqualFold(strings.TrimSpace(k), "q") {
			f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				continue
			}
			val = f
		}
		q[name] = val
	}
	best, bestQ := "", 0.0
	for _, enc := range []string{"br", "gzip", "deflate"} {
		v, ok := q[enc]
		if !ok {
			v, ok = q["*"]
		}
		if ok && v > bestQ {
			best, bestQ = enc, v
		}
	}
	return best
}

type compressWriter struct {
	http.ResponseWriter
	enc         string
	wroteHeader bool
	zw          io.WriteCloser
}

func (c *compressWriter) WriteHeader(code int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true
	h := c.Header()
	h.Add("Vary", "Accept-Encoding")
	if c.enc != "" && c.eligible(code) {
		h.Set("Content-Encoding", c.enc)
		h.Del("Content-Length")
		if etag := h.Get("Etag"); etag != "" && !strings.HasPrefix(etag, "W/") {
			h.Set("Etag", "W/"+etag)
		}
		c.zw = newEncoder(c.enc, c.ResponseWriter)
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *compressWriter) eligible(code int) bool {
	if code < 200 || code == http.StatusNoContent || code == http.StatusNotModified || code == http.StatusPartialContent {
		return false
	}
	h := c.Header()
	if h.Get("Content-Encoding") != "" || h.Get("Content-Range") != "" {
		return false
	}
	if n, err := strconv.ParseInt(h.Get("Content-Length"), 10, 64); err == nil && n < minCompressSize {
		return false
	}
	ct := h.Get("Content-Type")
	if mt, _, err := mime.ParseMediaType(ct); err == nil {
		ct = mt
	}
	for _, p := range compressibleTypes {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

func newEncoder(enc string, w io.Writer) io.WriteCloser {
	switch enc {
	case "br":
		return brotli.NewWriterLevel(w, 5)
	case "gzip":
		return gzip.NewWriter(w)
	default:
		zw, _ := flate.NewWriter(w, flate.DefaultCompression)
		return zw
	}
}

func (c *compressWriter) Write(b []byte) (int, error) {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", http.DetectContentType(b))
		}
		c.WriteHeader(http.StatusOK)
	}
	if c.zw != nil {
		return c.zw.Write(b)
	}
	return c.ResponseWriter.Write(b)
}

func (c *compressWriter) Flush() {
	if f, ok := c.zw.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	_ = http.NewResponseController(c.ResponseWriter).Flush()
}

func (c *compressWriter) close() {
	if c.zw != nil {
		_ = c.zw.Close()
	}
}

func (c *compressWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
