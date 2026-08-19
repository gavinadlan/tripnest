package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/repository"
)

// MinPasswordLength adalah panjang minimum password yang diterima.
const MinPasswordLength = 8

// Error yang dipetakan handler ke status HTTP tertentu. Handler memakai
// errors.Is, bukan pencocokan string, supaya pemetaannya tidak rapuh.
var (
	ErrEmailTaken         = errors.New("email sudah terdaftar")
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
	ErrValidation         = errors.New("data tidak valid")
)

type UserService interface {
	Register(ctx context.Context, req *model.RegisterUserRequest) (*model.User, error)
	Login(ctx context.Context, req *model.LoginUserRequest) (*model.AuthResponse, error)
	Profile(ctx context.Context, userID string) (*model.User, error)
	EnsureAdmin(ctx context.Context, email, password string) error
}

type userService struct {
	repo     repository.UserRepository
	cfg      *config.Config
	verifier *auth.Verifier
}

func NewUserService(repo repository.UserRepository, cfg *config.Config, verifier *auth.Verifier) UserService {
	return &userService{repo: repo, cfg: cfg, verifier: verifier}
}

func (s *userService) Register(ctx context.Context, req *model.RegisterUserRequest) (*model.User, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:     email,
		Password:  string(hashed),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Role:      auth.RoleUser,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		// Balapan antara cek di atas dan insert ini tetap mungkin terjadi;
		// unique index pada kolom email yang jadi penentu akhir.
		if repository.IsUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	user.Password = ""
	return user, nil
}

func (s *userService) Login(ctx context.Context, req *model.LoginUserRequest) (*model.AuthResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		// Jangan bocorkan bahwa formatnya yang salah; jawaban seragam
		// menyulitkan penebakan email yang terdaftar.
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Tetap jalankan satu perbandingan bcrypt supaya waktu respons
		// untuk email terdaftar dan tidak terdaftar tidak jauh berbeda.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.verifier.Sign(user.ID, user.Role, s.cfg.JWTExpiresIn)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &model.AuthResponse{Token: token, User: *user}, nil
}

func (s *userService) Profile(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	user.Password = ""
	return user, nil
}

// dummyHash adalah hash bcrypt yang valid untuk password acak. Dipakai
// hanya agar Login menghabiskan waktu yang mirip saat email tidak ada.
var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", fmt.Errorf("%w: email wajib diisi", ErrValidation)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("%w: format email tidak valid", ErrValidation)
	}
	return email, nil
}

func validatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: password minimal %d karakter", ErrValidation, MinPasswordLength)
	}
	if len(password) > 72 {
		// bcrypt memotong diam-diam di 72 byte; tolak lebih awal daripada
		// menerima password yang sebagian isinya diabaikan.
		return fmt.Errorf("%w: password maksimal 72 karakter", ErrValidation)
	}
	return nil
}

// EnsureAdmin mem-bootstrap akun admin dari environment saat startup.
// Ini menggantikan kredensial admin hardcoded di frontend: admin sekarang
// user biasa dengan role 'admin', dan login lewat endpoint yang sama.
func (s *userService) EnsureAdmin(ctx context.Context, email, password string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := s.repo.UpsertAdmin(ctx, normalized, string(hashed)); err != nil {
		return err
	}
	return nil
}
