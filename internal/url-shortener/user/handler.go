package user

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/internal/auth-session/user"
	"url-shorten/pkg"
)

type UserShortenerHandler struct {
	userSvc              user.UserService
	defaultRefreshExpiry time.Duration
	shortRefreshExpiry   time.Duration
	secureCookies        bool
}

func NewUserHandler(
	userSvc user.UserService,
	defaultRefreshExpiry time.Duration,
	shortRefreshExpiry time.Duration,
	secureCookies bool,
) *UserShortenerHandler {
	return &UserShortenerHandler{
		userSvc:              userSvc,
		defaultRefreshExpiry: defaultRefreshExpiry,
		shortRefreshExpiry:   shortRefreshExpiry,
		secureCookies:        secureCookies,
	}
}

func (h *UserShortenerHandler) setRefreshTokenCookie(c *gin.Context, token string, rememberMe bool) {
	expiryDuration := h.defaultRefreshExpiry
	if !rememberMe {
		expiryDuration = h.shortRefreshExpiry
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		token,
		int(expiryDuration/time.Second),
		"/v1/url-shortener/auth",
		"",
		h.secureCookies,
		true,
	)
}

func (h *UserShortenerHandler) setSessionCookie(c *gin.Context, sessionID string, rememberMe bool) {
	expiryDuration := h.defaultRefreshExpiry
	if !rememberMe {
		expiryDuration = h.shortRefreshExpiry
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"session_id",
		sessionID,
		int(expiryDuration/time.Second),
		"/v1/url-shortener",
		"",
		h.secureCookies,
		true,
	)
}

func (h *UserShortenerHandler) clearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/v1/url-shortener/auth",
		"",
		h.secureCookies,
		true,
	)
}

func (h *UserShortenerHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"session_id",
		"",
		-1,
		"/v1/url-shortener",
		"",
		h.secureCookies,
		true,
	)
}

func httpError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkg.ErrNotFound),
		errors.Is(err, pkg.ErrSessionNotFound):
		pkg.ErrorResponse(c, http.StatusNotFound, pkg.ErrNotFound)
	case errors.Is(err, pkg.ErrInvalidCredentials),
		errors.Is(err, pkg.ErrInvalidPassword),
		errors.Is(err, pkg.ErrTokenRevoked),
		errors.Is(err, pkg.ErrTokenInvalid),
		errors.Is(err, pkg.ErrRefreshTokenInvalid),
		errors.Is(err, pkg.ErrTokenExpired):
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrInvalidCredentials)
	case errors.Is(err, pkg.ErrInvalidInput):
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
	case errors.Is(err, pkg.ErrAlreadyExists),
		errors.Is(err, pkg.ErrRefreshTokenConcurrent):
		pkg.ErrorResponse(c, http.StatusConflict, pkg.ErrAlreadyExists)
	case errors.Is(err, pkg.ErrForbidden):
		pkg.ErrorResponse(c, http.StatusForbidden, pkg.ErrForbidden)
	default:
		pkg.ErrorResponse(c, http.StatusInternalServerError, pkg.ErrInternal)
	}
}

// @Summary      Register New User
// @Description  Register New User to database
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body user.RegisterUserRequest true "Register User Request"
// @Success      201 {object} pkg.Response "Register User Success"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /auth/register [post]
func (h *UserShortenerHandler) RegisterUser(c *gin.Context) {
	var req user.RegisterUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.Role = domain.User

	if err := h.userSvc.RegisterUser(c.Request.Context(), req); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "register user success", nil)
}

// @Summary      Update My Profile
// @Description  Update My Profile data to database
// @Tags         Users Authorized
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body user.UpdateUserRequest true "Update Profile Request"
// @Success      200 {object} UserProfileSwaggerResponse "Updated Profile Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me [patch]
func (h *UserShortenerHandler) UpdateMyProfile(c *gin.Context) {
	role := c.GetString("role")

	var req user.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.ID = c.GetString("userID")

	user, err := h.userSvc.UpdateUser(c.Request.Context(), role, &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated profile successfully", map[string]interface{}{
		"user": user,
	})
}

// @Summary      Update Data User by UserID(sub) (Admin)
// @Description  Update Data User to database By UserID(sub)
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        sub path string true "UserID(sub)"
// @Param        request body user.UpdateUserRequest true "Update User Request"
// @Success      200 {object} UserProfileSwaggerResponse "Updated User Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub} [patch]
func (h *UserShortenerHandler) UpdateUser(c *gin.Context) {
	role := c.GetString("role")

	var req user.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.ID = c.Param("sub")

	user, err := h.userSvc.UpdateUser(c.Request.Context(), role, &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated user successfully", map[string]interface{}{
		"user": user,
	})
}

// @Summary      Get My Profile
// @Description  Get My Profile Data when Login
// @Tags         Users Authorized
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} UserProfileSwaggerResponse "Get Profile Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me [get]
func (h *UserShortenerHandler) GetMyProfile(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.userSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get profile successfully", map[string]interface{}{
		"user": user,
	})
}

// @Summary      Get User By UserID(sub) (Admin)
// @Description  Get User Data by UserID from database
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub path string true "User ID"
// @Success      200 {object} UserProfileSwaggerResponse "Get User Successfully"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub} [get]
func (h *UserShortenerHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("sub")

	user, err := h.userSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get user successfully", map[string]interface{}{
		"user": user,
	})
}

// @Summary      Get All Users with Paginated List (Admin)
// @Description  Get All Users Data from database with Paginated List
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        page query int false "Page (Default: 1)"
// @Param        limit query int false "Limit (Default: 10)"
// @Success      200 {object} UsersListSwaggerResponse "Get Users Successfully"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users [get]
func (h *UserShortenerHandler) GetUsers(c *gin.Context) {
	page, limit, err := pkg.ParsePagination(c, 10, 100)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	users, err := h.userSvc.GetUsers(c.Request.Context(), page, limit)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get users successfully", map[string]interface{}{
		"users": users,
	})
}

// @Summary      Delete My Profile
// @Description  Delete My Profile from database with Revoke Sessions, Access Token, and Refresh Token
// @Tags         Users Authorized
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} pkg.Response "Delete My Profile Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me [delete]
func (h *UserShortenerHandler) DeleteMyProfile(c *gin.Context) {
	userID := c.GetString("userID")

	if err := h.userSvc.DeleteUser(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	h.clearRefreshTokenCookie(c)
	h.clearSessionCookie(c)

	pkg.SuccessResponse(c, http.StatusOK, "deleted my profile successfully", nil)
}

// @Summary      Delete User Profile by UserID(sub) (Admin)
// @Description  Delete User Profile from database and Revoke Sessions by User ID(sub)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub path string true "User ID"
// @Success      200 {object} pkg.Response "Delete User Profile Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub} [delete]
func (h *UserShortenerHandler) DeleteUserProfile(c *gin.Context) {
	userID := c.Param("sub")

	if err := h.userSvc.DeleteUser(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted user profile successfully", nil)
}

// @Summary      Login User
// @Description  Login user and return Access Token with set Refresh Token in HTTP-Only Cookie
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        User-Agent header string false "User Agent Client"
// @Param        request body user.LoginRequest true "Login Request Payload"
// @Success      200 {object} LoginSwaggerResponse "Logged In Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      401 {object} pkg.Response "Invalid Credentials"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /auth/login [post]
func (h *UserShortenerHandler) LoginUser(c *gin.Context) {
	var req user.LoginRequest

	agent := c.GetHeader("User-Agent")
	ip := c.ClientIP()

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	user, token, sid, err := h.userSvc.LoginUser(c.Request.Context(), agent, ip, req)
	if err != nil {
		httpError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, token.RefreshToken, user.RememberMe)
	h.setSessionCookie(c, sid, user.RememberMe)

	pkg.SuccessResponse(c, http.StatusOK, "logged in successfully", map[string]interface{}{
		"user":         user,
		"access_token": token.AccessToken,
	})
}

// @Summary      Refresh Tokens and Re-new Session Expired
// @Description  Re-new Access Token, Refresh Token, Sessions Expired using Refresh Token from Cookie
// @Tags         Auth
// @Produce      json
// @Param        Cookie  header  string  true  "Cookies (refresh_token=xxx; session_id=yyy)"
// @Success      200 {object} RefreshTokenSwaggerResponse "Refresh Token Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized / Missing Cookie"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /auth/refresh [post]
func (h *UserShortenerHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrRefreshTokenInvalid)
		return
	}

	sessionID, err := c.Cookie("session_id")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrSessionNotFound)
		return
	}

	res, isRememberMe, err := h.userSvc.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		httpError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, res.RefreshToken, isRememberMe)
	h.setSessionCookie(c, sessionID, isRememberMe)

	pkg.SuccessResponse(c, http.StatusOK, "refresh token successfully", map[string]interface{}{
		"access_token": res.AccessToken,
	})
}

// @Summary      Logout User
// @Description  Delete Session and Cleaned Cookie refresh_token
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Param        Cookie  header  string  true  "Cookies (refresh_token=xxx; session_id=yyy)"
// @Success      200 {object} pkg.Response "Logged Out Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized / Missing Cookie"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /auth/logout [post]
func (h *UserShortenerHandler) LogoutUser(c *gin.Context) {
	accessToken := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	userID := c.GetString("userID")

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrRefreshTokenInvalid)
		return
	}

	sessionID, err := c.Cookie("session_id")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrSessionNotFound)
		return
	}

	if err := h.userSvc.LogoutUser(c.Request.Context(), accessToken, refreshToken, userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	h.clearRefreshTokenCookie(c)
	h.clearSessionCookie(c)

	pkg.SuccessResponse(c, http.StatusOK, "logged out successfully", nil)
}

// @Summary      Get Active My Sessions
// @Description  Get Active My Session from Login History where not Logged Out yet
// @Tags         User Sessions Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        Cookie header string true "Cookie session_id"
// @Success      200 {object} SessionsListSwaggerResponse "Get Active Sessions Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized / Missing Cookie"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me/sessions [get]
func (h *UserShortenerHandler) GetActiveSessionsMyProfile(c *gin.Context) {
	userID := c.GetString("userID")
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrSessionNotFound)
		return
	}

	res, err := h.userSvc.GetActiveSessions(c.Request.Context(), userID, sessionID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get active sessions successfully", map[string]interface{}{
		"session": res,
	})
}

// @Summary      Revoke Session My Specific Session
// @Description  Revoke/Delete Specific Session Where still Active or not Logged Out yet
// @Tags         User Sessions Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        sid path string true "SessionID(sid)"
// @Success      200 {object} pkg.Response "Revoked Session Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Session Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me/sessions/{sid} [delete]
func (h *UserShortenerHandler) RevokeSessionMyProfile(c *gin.Context) {
	userID := c.GetString("userID")
	sessionID := c.Param("sid")

	if err := h.userSvc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked session successfully", nil)
}

// @Summary      Revoke All Other My Sessions
// @Description  Revoke All Other active Session but keep this Session Active
// @Tags         User Sessions Authorized
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} pkg.Response "Revoked All Other Sessions Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me/sessions/others [delete]
func (h *UserShortenerHandler) RevokeAllOtherSessionsMyProfile(c *gin.Context) {
	userID := c.GetString("userID")
	sessionID := c.GetString("sessionID")

	if err := h.userSvc.RevokeAllOtherSessions(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked all other sessions successfully", nil)
}

// @Summary      Revoke All My Sessions
// @Description  Revoke All My Active Sessions including this Session (Force Logout All)
// @Tags         User Sessions Authorized
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} pkg.Response "Revoked All Sessions Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /users/me/sessions [delete]
func (h *UserShortenerHandler) RevokeAllSessionsMyProfile(c *gin.Context) {
	userID := c.GetString("userID")

	if err := h.userSvc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	h.clearSessionCookie(c)
	h.clearRefreshTokenCookie(c)

	pkg.SuccessResponse(c, http.StatusOK, "revoked all sessions successfully", nil)
}

// @Summary      Get Active Sessions User (Admin)
// @Description  Get Active Session By UserID(sub)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub path string true "UserID(sub)"
// @Success      200 {object} SessionsListSwaggerResponse "Get Active Sessions Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/sessions [get]
func (h *UserShortenerHandler) GetActiveSessionsUser(c *gin.Context) {
	userID := c.Param("sub")
	sessionID := ""

	res, err := h.userSvc.GetActiveSessions(c.Request.Context(), userID, sessionID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get active sessions successfully", map[string]interface{}{
		"session": res,
	})
}

// @Summary      Revoke Session Specific User (Admin)
// @Description  Revoke Specific Session User by UserID(sub) and SessionID(sid)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub path string true "UserID(sub)"
// @Param        sid path string true "SessionID(sid)"
// @Success      200 {object} pkg.Response "Revoked Session Successfully"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      404 {object} pkg.Response "User/Session Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/sessions/{sid} [delete]
func (h *UserShortenerHandler) RevokeSessionUser(c *gin.Context) {
	userID := c.Param("sub")
	sessionID := c.Param("sid")

	if err := h.userSvc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked session successfully", nil)
}

// @Summary      Revoke All Sessions User (Admin)
// @Description  Revoke all active sessions for the user ID (sub)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub path string true "User ID"
// @Success      200 {object} pkg.Response "Revoked All Sessions Successfully"
// @Failure      404 {object} pkg.Response "User Not Found"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/sessions [delete]
func (h *UserShortenerHandler) RevokeAllSessionsUser(c *gin.Context) {
	userID := c.Param("sub")

	if err := h.userSvc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked all sessions successfully", nil)
}

// @Summary      Send Heartbeat
// @Description  Track session activity using the active or inactive state
// @Tags         Auth
// @Produce      json
// @Param        Cookie header string false "Cookie session_id"
// @Param        request body user.HeartbeatRequest true "Heartbeat Request"
// @Success      200 {object} pkg.Response "Heartbeat Recorded"
// @Failure      400 {object} pkg.Response "Invalid Request Body"
// @Failure      401 {object} pkg.Response "Unauthorized / Missing Cookie"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /auth/heartbeat [post]
func (h *UserShortenerHandler) Heartbeat(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, pkg.ErrSessionNotFound)
		return
	}

	var req user.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	if err := h.userSvc.Heartbeat(c.Request.Context(), sessionID, req); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "heartbeat recorded", nil)
}
