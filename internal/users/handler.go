package users

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"url-shorten/pkg"
)

type UserHandler struct {
	userSvc              UserService
	defaultRefreshExpiry time.Duration
	shortRefreshExpiry   time.Duration
}

func NewUserHandler(
	userSvc              UserService,
	defaultRefreshExpiry time.Duration,
	shortRefreshExpiry   time.Duration,
) *UserHandler {
	return &UserHandler{
		userSvc:              userSvc,
		defaultRefreshExpiry: defaultRefreshExpiry,
		shortRefreshExpiry:   shortRefreshExpiry,
	}
}

// Internal Helper set Cookie

func (h *UserHandler) setRefreshTokenCookie(c *gin.Context, token string, rememberMe bool) {
	expiryDuration := h.defaultRefreshExpiry
	if !rememberMe {
		expiryDuration = h.shortRefreshExpiry
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		token,
		int(expiryDuration/time.Second),
		"/api/v1/auth",
		"",
		false,
		true,
	)
}

func (h *UserHandler) clearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		"",
		false,
		true,
	)
}

// Error Handle

func httpError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkg.ErrNotFound):
		pkg.ErrorResponse(c, http.StatusNotFound, err)
	case errors.Is(err, pkg.ErrInvalidCredentials):
		pkg.ErrorResponse(c, http.StatusUnauthorized, err)
	case errors.Is(err, pkg.ErrInvalidPassword),
		errors.Is(err, pkg.ErrInvalidInput):
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
	default:
		pkg.ErrorResponse(c, http.StatusInternalServerError, err)
	}
}

// User CRUD

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req RegisterUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := h.userSvc.RegisterUser(c.Request.Context(), req); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "register user success", nil)
}

func (h *UserHandler) UpdateMyProfile(c *gin.Context) {
	var req UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.ID = c.GetString("userID")

	user, err := h.userSvc.UpdateUser(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated profile successfully", map[string]interface{}{
		"user": user,
	})
}

func (h *UserHandler) UpdateMyPassword(c *gin.Context) {
	var req UpdatePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.ID = c.GetString("userID")

	if err := h.userSvc.UpdatePassword(c.Request.Context(), req); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated password successfully", nil)
}

func (h *UserHandler) GetMyProfile(c *gin.Context) {
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

func (h *UserHandler) DeleteMyProfile(c *gin.Context) {
	userID := c.GetString("userID")

	if err := h.userSvc.DeleteUser(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted profile successfully", nil)
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.userSvc.GetUsers(c.Request.Context())
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get users successfully", map[string]interface{}{
		"uses": users,
	})
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
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

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.ID = c.Param("sub")

	user, err := h.userSvc.UpdateUser(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated user successfully", map[string]interface{}{
		"user": user,
	})
}

func (h *UserHandler) UpdatePasswordUser(c *gin.Context) {
	var req UpdatePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.ID = c.Param("sub")

	if err := h.userSvc.UpdatePassword(c.Request.Context(), req); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated password successfully", nil)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("sub")

	if err := h.userSvc.DeleteUser(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted user successfully", nil)
}

// User Auth

func (h *UserHandler) LoginUser(c *gin.Context) {
	var req LoginRequest

	agent := c.GetHeader("User-Agent")
	ip    := c.ClientIP()

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	user, token, err := h.userSvc.LoginUser(c.Request.Context(), agent, ip, req)
	if err != nil {
		httpError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, token.RefreshToken, user.RememberMe)

	pkg.SuccessResponse(c, http.StatusOK, "logged in successfully", map[string]interface{}{
		"user":         user,
		"access_token": token.AccessToken,
	})
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, err)
		return
	}

	res, isRememberMe, err := h.userSvc.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		httpError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, res.RefreshToken, isRememberMe)

	pkg.SuccessResponse(c, http.StatusOK, "refresh token successfully", map[string]interface{}{
		"access_token": res.AccessToken,
	})
}

func (h *UserHandler) LogoutUser(c *gin.Context) {
	accessToken := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, err)
		return
	}

	if err := h.userSvc.LogoutUser(c.Request.Context(), accessToken, refreshToken); err != nil {
		httpError(c, err)
		return
	}

	h.clearRefreshTokenCookie(c)

	pkg.SuccessResponse(c, http.StatusOK, "logged out successfully", nil)
}

// User Session

func (h *UserHandler) GetActiveMyProfileSessions(c *gin.Context) {
	userID := c.GetString("userID")

	res, err := h.userSvc.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get active profile sessions successfully", map[string]interface{}{
		"session": res,
	})
}

func (h *UserHandler) RevokeMyProfileSession(c *gin.Context) {
	userID    := c.GetString("userID")
	sessionID := c.Param("sid")

	if err := h.userSvc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked profile session successfully", nil)
}

func (h *UserHandler) RevokeAllMyProfileOtherSessions(c *gin.Context) {
	userID    := c.GetString("userID")
	sessionID := c.GetString("sessionID")

	if err := h.userSvc.RevokeAllOtherSessions(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked all profile other sessions successfully", nil)
}

func (h *UserHandler) RevokeAllMyProfileSessions(c *gin.Context) {
	userID := c.GetString("userID")

	if err := h.userSvc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked all profile sessions successfully", nil)
}

func (h *UserHandler) GetActiveSessionsUser(c *gin.Context) {
	userID := c.Param("sub")

	res, err := h.userSvc.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get active user sessions successfully", map[string]interface{}{
		"session": res,
	})
}

func (h *UserHandler) RevokeSessionUser(c *gin.Context) {
	userID    := c.Param("sub")
	sessionID := c.Param("sid")

	if err := h.userSvc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked user session successfully", nil)
}

func (h *UserHandler) RevokeAllSessionsUser(c *gin.Context) {
	userID := c.Param("sub")

	if err := h.userSvc.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "revoked all user sessions successfully", nil)
}