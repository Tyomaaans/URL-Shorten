package users

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"url-shorten/internal/domains"
	"url-shorten/internal/sessions"
	"url-shorten/internal/tokens"
	"url-shorten/pkg"

	jsonValidator "url-shorten/pkg"
)

// Interface

type UserService interface {
	// User CRUD
	RegisterUser(ctx context.Context, req RegisterUserRequest) error
	UpdateUser(ctx context.Context, update *UpdateUserRequest) (*UserResponse, error)
	UpdatePassword(ctx context.Context, update UpdatePasswordRequest) error
	GetUsers(ctx context.Context) ([]UserResponse, error)
	GetUserByID(ctx context.Context, rawUserID string) (*UserResponse, error)
	DeleteUser(ctx context.Context, rawUserID string) error
	
	// User Auth
	LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *tokens.TokenPairResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*tokens.TokenPairResponse, bool, error)
	LogoutUser(ctx context.Context, accessToken, refreshToken string) error

	// User Session
	GetActiveSessions(ctx context.Context, rawUserID string) ([]sessions.SessionResponse, error)
	RevokeSession(ctx context.Context, rawUserID, rawSessionID string) error
	RevokeAllOtherSessions(ctx context.Context, rawUserID, rawExceptSessionID string) error
	RevokeAllSessions(ctx context.Context, rawUserID string) error
}

// Implementation

type userService struct {
	userRepo   domains.UserRepository
	tokenSvc   tokens.JWTService
	sessionSvc sessions.SessionService
	validate   *validator.Validate
}

func NewUserService (
	userRepo   domains.UserRepository,
	tokenSvc   tokens.JWTService,
	sessionSvc sessions.SessionService,
	validate   *validator.Validate,
) UserService {
	return &userService{
		userRepo:   userRepo,
		tokenSvc:   tokenSvc,
		sessionSvc: sessionSvc,
		validate:   validate,
	}
}

// User CRUD

func (s *userService) RegisterUser(ctx context.Context, req RegisterUserRequest) error {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return fmt.Errorf("%w %v", pkg.ErrInvalidInput, err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return pkg.ErrInternal
	}

	id          := uuid.NewString()
	req.Password = string(hashedPassword)

	payload := ToRegisterUserEntity(id, req)

	if err := s.userRepo.CreateUser(ctx, payload); err != nil {
		return err
	}

	return nil
}

func (s *userService) UpdateUser(ctx context.Context, update *UpdateUserRequest) (*UserResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, update); err != nil {
		return nil, fmt.Errorf("%w %v", pkg.ErrInvalidInput, err)
	}

	userID, err := parseOrDecodeUUID(update.ID)
    if err != nil {
        return nil, err
    }

	update.ID = userID

	if update.RememberMe != nil {
		update.RememberMe = nil
	}

	user, err := s.userRepo.UpdateUser(ctx, ToUpdateUserEntity(update))
	if err != nil {
		return nil, err
	}

	res, err := ToUserResponse(user)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *userService) UpdatePassword(ctx context.Context, update UpdatePasswordRequest) error {
	if err := jsonValidator.ValidateStruct(s.validate, update); err != nil {
		return fmt.Errorf("%w %v", pkg.ErrInvalidInput, err)
	}

	userID, err := parseOrDecodeUUID(update.ID)
    if err != nil {
        return err
    }

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	oldPassword, _, err := s.userRepo.GetPasswordByEmail(ctx, user.Email)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(update.OldPassword)); err != nil {
		return pkg.ErrInvalidPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(update.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return pkg.ErrInternal
	}

	update.NewPassword = string(hashedPassword)

	if err := s.userRepo.UpdatePassword(ctx, ToUpdatePasswordEntity(update)); err != nil {
		return err
	}

	return nil
}

func (s *userService) GetUsers(ctx context.Context) ([]UserResponse, error) {
	users, err := s.userRepo.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	res, err := ToUserListResponse(users)
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

	res, err := ToUserResponse(user)
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

	if err := s.userRepo.DeleteUser(ctx, userID); err != nil {
		return err
	}

	return nil
}

// User Auth

func (s *userService) LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *tokens.TokenPairResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return nil, nil, fmt.Errorf("%w %v", pkg.ErrInvalidInput, err)
	}

	hashedPassword, userID, err := s.userRepo.GetPasswordByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, pkg.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return nil, nil, pkg.ErrInvalidCredentials
	}

	user, err := s.userRepo.UpdateUser(ctx, &domains.UpdateUserEntity{
		ID:         userID,
		RememberMe: &req.RememberMe,
	})
	if err != nil {
		return nil, nil, err
	}

	tokenPair, err := s.tokenSvc.GenerateTokenPair(ctx, user.ID, agent, ip, user.RememberMe)
	if err != nil {
		return nil, nil, err
	}

	res, err := ToUserResponse(user)
	if err != nil {
		return nil, nil, err
	}

	return res, tokens.ToTokenPairResponse(*tokenPair), nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*tokens.TokenPairResponse, bool, error) {
	tokenPair, err := s.tokenSvc.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, false, err
	}

	user, err := s.userRepo.GetUserByID(ctx, tokenPair.UserID)
	if err != nil {
		return nil, false, err
	}

	return tokens.ToTokenPairResponse(*tokenPair), user.RememberMe, nil
}

func (s *userService) LogoutUser(ctx context.Context, accessToken, refreshToken string) error {
	if err := s.tokenSvc.RevokeTokens(ctx, accessToken, refreshToken); err != nil {
		return err
	}

	return nil
}

// User Session

func (s *userService) GetActiveSessions(ctx context.Context, rawUserID string) ([]sessions.SessionResponse, error) {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return nil, err
    }

    session, err := s.sessionSvc.GetActiveSessions(ctx, userID)
    if err != nil {
        return nil, err
    }

    res := sessions.ToSessionListResponse(session)
    for i := range session {
        uID, err := uuidToBase64(session[i].UserID)
        if err != nil {
            return nil, err
        }

        sID, err := uuidToBase64(session[i].SessionID)
        if err != nil {
            return nil, err
        }

        res[i].UserID    = uID
        res[i].SessionID = sID
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

    if err := s.sessionSvc.RevokeAllSessions(ctx, userID); err != nil {
        return err
    }

    return nil
}

// Internal Helper

func parseOrDecodeUUID(input string) (string, error) {
    if input == "" {
        return "", pkg.ErrInvalidInput
    }

    if _, err := uuid.Parse(input); err == nil {
        return input, nil
    }

    decoded, err := base64.RawURLEncoding.DecodeString(input)
    if err != nil {
        return "", pkg.ErrInvalidInput
    }

    parsed, err := uuid.FromBytes(decoded)
    if err != nil {
        return "", pkg.ErrInvalidInput
    }

    return parsed.String(), nil
}