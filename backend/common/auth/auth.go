// Package auth menyediakan verifikasi JWT bersama untuk seluruh service TripNest.
//
// Semua service memverifikasi token dengan secret yang sama (JWT_SECRET) dan
// membaca identitas pemanggil dari context, bukan dari body request.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/golang-jwt/jwt/v5"
)

// MinSecretLength adalah panjang minimum JWT_SECRET yang diterima.
// Nilai pendek seperti "dev-secret" mudah ditebak dan membuat siapa pun
// bisa menandatangani token admin sendiri.
const MinSecretLength = 32

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
	// RoleService dipakai untuk token yang diterbitkan satu service untuk
	// memanggil service lain, bukan untuk manusia. Token ini berumur pendek
	// dan tidak pernah sampai ke browser.
	RoleService = "service"
)

var (
	ErrSecretMissing = fmt.Errorf("JWT_SECRET wajib diisi dan minimal %d karakter", MinSecretLength)
	ErrNoToken       = errors.New("token tidak ditemukan")
	ErrInvalidToken  = errors.New("token tidak valid")
	ErrForbiddenRole = errors.New("akses ditolak")
)

// Claims adalah isi token TripNest. Dipakai bersama oleh penerbit
// (user-service) dan semua verifikator.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type ctxKey struct{}

var claimsCtxKey ctxKey

// Verifier memverifikasi dan menerbitkan token TripNest.
type Verifier struct {
	secret []byte
	issuer string
}

// NewVerifier menolak secret kosong atau terlalu pendek. Panggil ini saat
// startup dan hentikan proses bila gagal: service yang jalan dengan secret
// lemah lebih berbahaya daripada service yang tidak jalan.
func NewVerifier(secret string) (*Verifier, error) {
	if len(strings.TrimSpace(secret)) < MinSecretLength {
		return nil, ErrSecretMissing
	}
	return &Verifier{secret: []byte(secret), issuer: "user-service"}, nil
}

// Sign menerbitkan token untuk seorang user.
func (v *Verifier) Sign(userID, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    v.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(v.secret)
}

// Parse memvalidasi tanda tangan, masa berlaku, dan algoritma token.
func (v *Verifier) Parse(raw string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (interface{}, error) {
		// Tolak token yang mengaku memakai algoritma lain (mis. "none" atau RS256
		// dengan public key sebagai HMAC key).
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritma tidak didukung: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UserID) == "" {
		return nil, ErrInvalidToken
	}
	if claims.Role == "" {
		claims.Role = RoleUser
	}
	return &claims, nil
}

// Required menolak request tanpa token yang sah.
func (v *Verifier) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := v.fromRequest(r)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

// Optional menaruh claims di context bila ada, tapi tidak menolak request
// tanpa token. Berguna untuk endpoint publik yang perilakunya berbeda
// saat pemanggil sudah login.
func (v *Verifier) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, err := v.fromRequest(r); err == nil {
			r = r.WithContext(WithClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin menolak pemanggil yang bukan admin. Selalu pasang setelah
// Required, bukan menggantikannya.
func (v *Verifier) RequireAdmin(next http.Handler) http.Handler {
	return v.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			utils.WriteError(w, http.StatusForbidden, ErrForbiddenRole)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireServiceOrAdmin mengizinkan panggilan antar-service dan admin,
// tapi bukan user biasa. Dipakai untuk endpoint internal seperti
// pengambilan detail booking oleh payment-service.
func (v *Verifier) RequireServiceOrAdmin(next http.Handler) http.Handler {
	return v.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := FromContext(r.Context())
		if claims.Role != RoleService && claims.Role != RoleAdmin {
			utils.WriteError(w, http.StatusForbidden, ErrForbiddenRole)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// ServiceToken menerbitkan token antar-service berumur pendek.
func (v *Verifier) ServiceToken(serviceName string) (string, error) {
	return v.Sign("svc:"+serviceName, RoleService, 5*time.Minute)
}

// AdminOnly membungkus satu handler. Bentuk ini dipakai service yang
// memakai http.ServeMux, yang tidak punya middleware per-route.
func (v *Verifier) AdminOnly(h http.HandlerFunc) http.Handler {
	return v.RequireAdmin(h)
}

// AuthedOnly membungkus satu handler dengan syarat token sah.
func (v *Verifier) AuthedOnly(h http.HandlerFunc) http.Handler {
	return v.Required(h)
}

func (v *Verifier) fromRequest(r *http.Request) (*Claims, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, ErrNoToken
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, ErrNoToken
	}
	return v.Parse(strings.TrimSpace(parts[1]))
}

// WithClaims menaruh claims ke dalam context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsCtxKey, claims)
}

// FromContext mengambil claims yang sudah divalidasi middleware.
func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsCtxKey).(*Claims)
	return claims, ok && claims != nil
}

// UserID mengembalikan id pemanggil. Inilah satu-satunya sumber identitas
// yang boleh dipercaya -- jangan pernah membaca user_id dari body request.
func UserID(ctx context.Context) (string, bool) {
	claims, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

// IsAdmin melaporkan apakah pemanggil punya role admin.
func IsAdmin(ctx context.Context) bool {
	claims, ok := FromContext(ctx)
	return ok && claims.Role == RoleAdmin
}
