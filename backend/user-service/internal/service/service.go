package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/gavinadlan/tripnest/backend/user-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/repository"
)

type UserService interface {
	Register(ctx context.Context, req *model.RegisterUserRequest) (*model.User, error)
	Login(ctx context.Context, req *model.LoginUserRequest) (*model.AuthResponse, error)
}

type userService struct {
	repo repository.UserRepository
	cfg  *config.Config
}

func NewUserService(repo repository.UserRepository, cfg *config.Config) UserService {
	return &userService{repo: repo, cfg: cfg}
}

func (s *userService) Register(ctx context.Context, req *model.RegisterUserRequest) (*model.User, error) {
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:     req.Email,
		Password:  string(hashed),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "user",
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return user, nil
}

func (s *userService) Login(ctx context.Context, req *model.LoginUserRequest) (*model.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	claims := model.Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiresIn)),
			Issuer:    "user-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	user.Password = ""
	return &model.AuthResponse{
		Token: tokenString,
		User:  *user,
	}, nil
}
