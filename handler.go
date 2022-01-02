package lrucache

import (
	"bytes"
	"net/http"
	"time"
)

// HttpHandler is can be used as HTTP Middleware in order to cache requests,
// for example static assets. The request's raw URI is used and nothing else,
// so this handler should not be used to cache requests with responses depending
// on cookies or request headers.
type HttpHandler struct {
	cache      *Cache
	fetcher    http.Handler
	defaultTTL time.Duration
}

var _ http.Handler = (*HttpHandler)(nil)

type cachedResponseWriter struct {
	w          http.ResponseWriter
	statusCode int
	buf        bytes.Buffer
}

type cachedResponse struct {
	headers    http.Header
	statusCode int
	data       []byte
	fetched    time.Time
}

var _ http.ResponseWriter = (*cachedResponseWriter)(nil)

func (crw *cachedResponseWriter) Header() http.Header {
	return crw.w.Header()
}

func (crw *cachedResponseWriter) Write(bytes []byte) (int, error) {
	return crw.buf.Write(bytes)
}

func (crw *cachedResponseWriter) WriteHeader(statusCode int) {
	crw.statusCode = statusCode
	crw.w.WriteHeader(statusCode)
}

// Returns a new caching HttpHandler. If no entry in the cache is found or it was too old, `fetcher` is called with
// a modified http.ResponseWriter and the response is stored in the cache. If `fetcher` sets the "Expires" header,
// the ttl is set appropriately (otherwise, the default ttl passed as argument here is used).
// `maxmemory` should be in the unit bytes.
func NewHttpHandler(maxmemory int, ttl time.Duration, fetcher http.Handler) *HttpHandler {
	return &HttpHandler{
		cache:      New(maxmemory),
		defaultTTL: ttl,
		fetcher:    fetcher,
	}
}

// Tries to serve a response to r from cache or calls next and stores the response to the cache for the next time.
func (h *HttpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.ServeHTTP(rw, r)
		return
	}

	cr := h.cache.Get(r.RequestURI, func() (interface{}, time.Duration, int) {
		crw := &cachedResponseWriter{
			w:          rw,
			statusCode: 200,
			buf:        bytes.Buffer{},
		}

		h.fetcher.ServeHTTP(crw, r)

		cr := &cachedResponse{
			headers:    rw.Header(),
			statusCode: crw.statusCode,
			data:       crw.buf.Bytes(),
			fetched:    time.Now(),
		}

		ttl := h.defaultTTL
		if cr.headers.Get("Expires") != "" {
			if expires, err := http.ParseTime(cr.headers.Get("Expires")); err != nil {
				ttl = time.Until(expires)
			}
		}

		return cr, ttl, len(cr.data)
	}).(*cachedResponse)

	for key, val := range cr.headers {
		rw.Header()[key] = val
	}

	rw.WriteHeader(cr.statusCode)
	rw.Write(cr.data)
}
