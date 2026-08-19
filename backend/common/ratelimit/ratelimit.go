// Package ratelimit menyediakan pembatas laju in-memory sederhana.
//
// Cukup untuk melindungi endpoint login dan register dari brute force pada
// deployment satu instance. Untuk banyak instance, ganti backend-nya dengan
// Redis -- antarmuka Middleware tidak perlu berubah.
package ratelimit

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/utils"
)

var errTooMany = errors.New("terlalu banyak percobaan, coba lagi nanti")

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter membatasi jumlah request per kunci (default: alamat IP klien)
// memakai token bucket.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	refill   float64 // token per detik
	ttl      time.Duration
	stop     chan struct{}
}

// New membuat limiter yang mengizinkan burst sebanyak `capacity` request
// dan mengisi ulang `capacity` token setiap `window`.
func New(capacity int, window time.Duration) *Limiter {
	if capacity < 1 {
		capacity = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	l := &Limiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(capacity),
		refill:   float64(capacity) / window.Seconds(),
		ttl:      window * 10,
		stop:     make(chan struct{}),
	}
	go l.janitor()
	return l
}

// Close menghentikan goroutine pembersih.
func (l *Limiter) Close() { close(l.stop) }

// Allow melaporkan apakah satu request untuk `key` boleh diproses.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.capacity - 1, lastSeen: now}
		return true
	}

	b.tokens += now.Sub(b.lastSeen).Seconds() * l.refill
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware menolak request berlebih dengan 429.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(ClientIP(r)) {
			w.Header().Set("Retry-After", "60")
			utils.WriteError(w, http.StatusTooManyRequests, errTooMany)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP mengambil alamat klien, menghormati X-Forwarded-For bila ada
// di belakang proxy.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, found := strings.Cut(xff, ","); found {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (l *Limiter) janitor() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for key, b := range l.buckets {
				if now.Sub(b.lastSeen) > l.ttl {
					delete(l.buckets, key)
				}
			}
			l.mu.Unlock()
		}
	}
}
