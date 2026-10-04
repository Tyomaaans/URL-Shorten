package user

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/internal/auth-session/session"
	"url-shorten/internal/auth-session/token"
	"url-shorten/pkg"

	jsonValidator "url-shorten/pkg"
)

// Interface

type UserService interface {
	RegisterUser(ctx context.Context, req RegisterUserRequest) error
	UpdateUser(ctx context.Context, role string, update *UpdateUserRequest) (*UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*UserResponse, error)
	GetUsers(ctx context.Context, page, limit int) (*PaginatedUserResponse, error)
	DeleteUser(ctx context.Context, rawUserID string) error

	LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *token.TokenPairResponse, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (*token.TokenPairResponse, bool, error)
	LogoutUser(ctx context.Context, accessToken, refreshToken, rawUserID, rawSessionID string) error

	GetActiveSessions(ctx context.Context, rawUserID, rawSessionID string) ([]session.SessionResponse, error)
	RevokeSession(ctx context.Context, rawUserID, rawSessionID string) error
	RevokeAllOtherSessions(ctx context.Context, urawUserID, rawExceptSessionID string) error
	RevokeAllSessions(ctx context.Context, rawUserID string) error

	Heartbeat(ctx context.Context, rawSessionID string, req HeartbeatRequest) error
}

// Implementation

type userService struct {
	userRepo   domain.UserRepository
	tokenSvc   token.JWTService
	sessionSvc session.SessionService
	validate   *validator.Validate
}

func NewUserService(
	userRepo   domain.UserRepository,
	tokenSvc   token.JWTService,
	sessionSvc session.SessionService,
	validate   *validator.Validate,
) UserService {
	return &userService{
		userRepo:   userRepo,
		tokenSvc:   tokenSvc,
		sessionSvc: sessionSvc,
		validate:   validate,
	}
}

func (s *userService) RegisterUser(ctx context.Context, req RegisterUserRequest) error {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return pkg.ErrInternal
	}

	id := uuid.NewString()
	req.Password = string(hashedPassword)

	payload := ToRegisterUserEntity(id, req)

	if err := s.userRepo.CreateUser(ctx, payload); err != nil {
		return err
	}

	return nil
}

func (s *userService) UpdateUser(ctx context.Context, role string, update *UpdateUserRequest) (*UserResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, update); err != nil {
		return nil, fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	if role == string(domain.Admin) {
		return nil, pkg.ErrForbidden
	}

	if update.RememberMe != nil {
		update.RememberMe = nil
	}

	if err := s.userRepo.UpdateUser(ctx, ToUpdateUserEntity(update)); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	res, err := ToUserResponse(*user)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *userService) GetUserByID(ctx context.Context, rawUserID string) (*UserResponse, error) {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	res, err := ToUserResponse(*user)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *userService) GetUsers(ctx context.Context, page, limit int) (*PaginatedUserResponse, error) {
	users, pages, err := s.userRepo.GetUsers(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	res, err := ToPaginatedUserResponse(users, page, limit, pages)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *userService) DeleteUser(ctx context.Context, rawUserID string) error {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if string(user.Role) == string(domain.Admin) && user.ID == userID {
		return pkg.ErrForbidden
	}

	if err := s.userRepo.DeleteUser(ctx, userID); err != nil {
		return err
	}

	if n, err := s.tokenSvc.RevokeAllUserTokens(ctx, userID); err != nil {
		log.Printf("revoke tokens for user %s failed (revoked %d): %v", userID, n, err)
	}

	return nil
}

func (s *userService) LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *token.TokenPairResponse, string, error) {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return nil, nil, "", fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	hashedPassword, userID, err := s.userRepo.GetPasswordByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, "", pkg.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return nil, nil, "", pkg.ErrInvalidCredentials
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, "", pkg.ErrInvalidCredentials
	}

	user.RememberMe = req.RememberMe
	tokenPair, sid, err := s.tokenSvc.GenerateTokenPair(ctx, user.ID, agent, ip, string(user.Role), req.RememberMe)
	if err != nil {
		return nil, nil, "", err
	}

	res, err := ToUserResponse(*user)
	if err != nil {
		return nil, nil, "", err
	}

	rawSessionID, err := uuidToBase64(sid)
	if err != nil {
		return nil, nil, "", err
	}

	return res, token.ToTokenPairResponse(*tokenPair), rawSessionID, nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*token.TokenPairResponse, bool, error) {
	tokenPair, err := s.tokenSvc.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, false, err
	}

	return token.ToTokenPairResponse(*tokenPair), tokenPair.RememberMe, nil
}

func (s *userService) LogoutUser(ctx context.Context, accessToken, refreshToken, rawUserID, rawSessionID string) error {
	if err := s.tokenSvc.RevokeTokens(ctx, accessToken, refreshToken); err != nil {
		return err
	}

	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return err
	}

	sessionID, err := parseOrDecodeUUID(rawSessionID)
	if err != nil {
		return err
	}

	if err := s.sessionSvc.RevokeSession(ctx, userID, sessionID); err != nil {
		return err
	}

	return nil
}

func (s *userService) GetActiveSessions(ctx context.Context, rawUserID, rawSessionID string) ([]session.SessionResponse, error) {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return nil, err
	}

	var sid string
	if rawSessionID != "" {
		sessionID, err := parseOrDecodeUUID(rawSessionID)
		if err != nil {
			return nil, err
		}
		sid = sessionID
	}

	sessions, err := s.sessionSvc.GetActiveSessions(ctx, userID, sid)
	if err != nil {
		return nil, err
	}

	res, err := session.ToSessionListResponse(sessions)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *userService) RevokeSession(ctx context.Context, rawUserID, rawSessionID string) error {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return err
	}

	sessionID, err := parseOrDecodeUUID(rawSessionID)
	if err != nil {
		return err
	}

	if err := s.sessionSvc.RevokeSession(ctx, userID, sessionID); err != nil {
		return err
	}

	return nil
}

func (s *userService) RevokeAllOtherSessions(ctx context.Context, rawUserID, rawExceptSessionID string) error {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return err
	}

	exceptSessionID, err := parseOrDecodeUUID(rawExceptSessionID)
	if err != nil {
		return err
	}

	if err := s.sessionSvc.RevokeAllOtherSessions(ctx, userID, exceptSessionID); err != nil {
		return err
	}

	return nil
}

func (s *userService) RevokeAllSessions(ctx context.Context, rawUserID string) error {
	userID, err := parseOrDecodeUUID(rawUserID)
	if err != nil {
		return err
	}

	if n, err := s.tokenSvc.RevokeAllUserTokens(ctx, userID); err != nil {
		log.Printf("revoke tokens for user %s failed (revoked %d): %v", userID, n, err)
	}

	return nil
}

func (s *userService) Heartbeat(ctx context.Context, rawSessionID string, req HeartbeatRequest) error {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	sessionID, err := parseOrDecodeUUID(rawSessionID)
	if err != nil {
		return err
	}

	if err := s.sessionSvc.Heartbeat(ctx, sessionID, req.State); err != nil {
		return err
	}

	return nil
}

// Internal Helper

func parseOrDecodeUUID(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("%w%s", pkg.ErrInvalidInput, "user_id id or session_id is required")
	}

	if _, err := uuid.Parse(input); err == nil {
		return input, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return "", fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	parsed, err := uuid.FromBytes(decoded)
	if err != nil {
		return "", fmt.Errorf("%w%s", pkg.ErrInvalidInput, err)
	}

	return parsed.String(), nil
}
