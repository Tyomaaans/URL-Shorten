package user

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/internal/auth-session/session"
	"url-shorten/internal/auth-session/token"
)

const testUserID = "d89ef3c5-d14a-45d7-a821-8a58f9305d4d"

type userRepoStub struct {
	user           domain.UserEntity
	password       string
	getUserByIDCnt int
}

func (s *userRepoStub) CreateUser(context.Context, *domain.CreateUserEntity) error { return nil }
func (s *userRepoStub) UpdateUser(context.Context, *domain.UpdateUserEntity) error { return nil }
func (s *userRepoStub) GetUsers(context.Context, int, int) ([]domain.UserEntity, int64, error) {
	return nil, 0, nil
}
func (s *userRepoStub) GetUserByID(context.Context, string) (*domain.UserEntity, error) {
	s.getUserByIDCnt++
	copy := s.user
	return &copy, nil
}
func (s *userRepoStub) GetPasswordByEmail(context.Context, string) (string, string, error) {
	return s.password, s.user.ID, nil
}
func (s *userRepoStub) DeleteUser(context.Context, string) error { return nil }

type tokenServiceStub struct {
	generatedRememberMe bool
	refreshed           *domain.TokenPairEntity
}

func (s *tokenServiceStub) GenerateTokenPair(_ context.Context, userID, _, _, _ string, rememberMe bool) (*domain.TokenPairEntity, string, error) {
	s.generatedRememberMe = rememberMe
	return &domain.TokenPairEntity{
		UserID: userID, AccessToken: "access", RefreshToken: "refresh", RememberMe: rememberMe,
	}, "5e7e717e-65e9-4cbc-b628-d4c60b490a41", nil
}
func (*tokenServiceStub) ValidateAccessToken(context.Context, string) (*domain.ClaimsEntity, error) {
	return nil, nil
}
func (s *tokenServiceStub) RefreshTokens(context.Context, string) (*domain.TokenPairEntity, error) {
	return s.refreshed, nil
}
func (*tokenServiceStub) RevokeTokens(context.Context, string, string) error       { return nil }
func (*tokenServiceStub) RevokeAllUserTokens(context.Context, string) (int, error) { return 0, nil }

type sessionServiceStub struct{}

func (*sessionServiceStub) CreateSession(context.Context, *domain.SessionEntity, time.Duration) error {
	return nil
}
func (*sessionServiceStub) IsSessionActive(context.Context, string) (bool, error) { return true, nil }
func (*sessionServiceStub) GetActiveSessions(context.Context, string, string) ([]domain.SessionEntity, error) {
	return nil, nil
}
func (*sessionServiceStub) RevokeSession(context.Context, string, string) error          { return nil }
func (*sessionServiceStub) RevokeAllOtherSessions(context.Context, string, string) error { return nil }
func (*sessionServiceStub) ExtendSession(context.Context, string, string, time.Duration) error {
	return nil
}
func (*sessionServiceStub) PurgeUserSessions(context.Context, string) (int, error) { return 0, nil }
func (*sessionServiceStub) Heartbeat(context.Context, string, string) error        { return nil }

func newUserServiceForTest(repo domain.UserRepository, tokenSvc token.JWTService) UserService {
	return NewUserService(repo, tokenSvc, session.SessionService(&sessionServiceStub{}), validator.New())
}

func TestLoginRememberMeIsScopedToTheRequestedSession(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Valid1!Password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		storedRememberMe  bool
		requestedRemember bool
	}{
		{name: "enable on a previously short account state", storedRememberMe: false, requestedRemember: true},
		{name: "disable on a previously remembered account state", storedRememberMe: true, requestedRemember: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userRepoStub{
				password: string(hash),
				user: domain.UserEntity{
					ID: testUserID, Name: "Test User", Email: "user@example.com",
					Role: domain.User, RememberMe: tt.storedRememberMe,
				},
			}
			tokens := &tokenServiceStub{}
			svc := newUserServiceForTest(repo, tokens)

			response, _, _, err := svc.LoginUser(context.Background(), "agent", "192.0.2.1", LoginRequest{
				Email: "user@example.com", Password: "Valid1!Password", RememberMe: tt.requestedRemember,
			})
			if err != nil {
				t.Fatal(err)
			}
			if tokens.generatedRememberMe != tt.requestedRemember {
				t.Fatalf("token remember_me = %v, want %v", tokens.generatedRememberMe, tt.requestedRemember)
			}
			if response.RememberMe != tt.requestedRemember {
				t.Fatalf("response remember_me = %v, want %v", response.RememberMe, tt.requestedRemember)
			}
		})
	}
}

func TestRefreshUsesSessionRememberMeWithoutReadingUserGlobalState(t *testing.T) {
	repo := &userRepoStub{user: domain.UserEntity{ID: testUserID, RememberMe: false}}
	tokens := &tokenServiceStub{refreshed: &domain.TokenPairEntity{
		UserID: testUserID, AccessToken: "access", RefreshToken: "refresh", RememberMe: true,
	}}
	svc := newUserServiceForTest(repo, tokens)

	response, rememberMe, err := svc.RefreshToken(context.Background(), "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if !rememberMe || !response.RememberMe {
		t.Fatalf("refresh lost session remember_me: handler=%v response=%v", rememberMe, response.RememberMe)
	}
	if repo.getUserByIDCnt != 0 {
		t.Fatalf("refresh read user-global state %d times", repo.getUserByIDCnt)
	}
}
