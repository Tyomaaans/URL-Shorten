package shortens

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"url-shorten/pkg"
)

type ShortenHandler struct {
	shortenSvc ShortenService
}

func NewShortenHandler(shortenSvc ShortenService) *ShortenHandler {
	return &ShortenHandler{
		shortenSvc: shortenSvc,
	}
}

// Error Handle

func httpError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkg.ErrNotFound):
		pkg.ErrorResponse(c, http.StatusNotFound, err)
	case errors.Is(err, pkg.ErrForbidden):
		pkg.ErrorResponse(c, http.StatusForbidden, err)
	case errors.Is(err, pkg.ErrInvalidInput):
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
	default:
		pkg.ErrorResponse(c, http.StatusInternalServerError, err)
	}
}

// Shorten Public

func (h *ShortenHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("code")

	originalURL, err := h.shortenSvc.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		httpError(c, err)
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func (h *ShortenHandler) CreateShorten(c *gin.Context) {
	var req CreateShortenPublicRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	shorten, err := h.shortenSvc.CreateShortenPublic(c.Request.Context(), req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "create short url successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// Shorten Authorized

func (h *ShortenHandler) CreateMyShorten(c *gin.Context) {
	var req CreateShortenAuthorizedRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.Owner = c.GetString("userID")

	shorten, err := h.shortenSvc.CreateShortenAuthorized(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "create short url successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) GetMyShortens(c *gin.Context) {
	ownerID := c.GetString("userID")

	shortens, err := h.shortenSvc.GetShortenByOwner(c.Request.Context(), ownerID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get my shortens successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

func (h *ShortenHandler) GetMyShortenByID(c *gin.Context) {
	shortenID := c.Param("shid")
	userID    := c.GetString("userID")

	shorten, err := h.shortenSvc.GetShortenByID(c.Request.Context(), shortenID, userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get user shorten successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) UpdateMyShorten(c *gin.Context) {
	var req UpdateShortenAuthorizedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.ID    = c.Param("shid")
	req.Owner = c.GetString("userID")

	shorten, err := h.shortenSvc.UpdateShorten(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated my shorten successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) SetURLStatus(c *gin.Context) {
	var req UpdateShortenAuthorizedRequest
	
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID    = c.Param("shid")
	req.Owner = c.GetString("userID")

	shorten, err := h.shortenSvc.SetURLStatus(c.Request.Context(), req.Owner, req.ID, *req.IsActive)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "set url status shorten successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) DeleteMyShorten(c *gin.Context) {
	userID    := c.GetString("userID")
	shortenID := c.Param("shid")

	if err := h.shortenSvc.DeleteShorten(c.Request.Context(), shortenID, userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted my shorten successfully", nil)
}

func (h *ShortenHandler) GetUserShortens(c *gin.Context) {
	shortens, err := h.shortenSvc.GetShortens(c.Request.Context())
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get user shortens successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

func (h *ShortenHandler) GetUserShortenByID(c *gin.Context) {
	userID    := c.Param("sub")
	shortenID := c.Param("shid")

	shorten, err := h.shortenSvc.GetShortenByID(c.Request.Context(), shortenID, userID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get user shorten successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) GetShortensByUserID(c *gin.Context) {
	ownerID := c.Param("sub")

	shortens, err := h.shortenSvc.GetShortenByOwner(c.Request.Context(), ownerID)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get user shortens successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

func (h *ShortenHandler) UpdateUserShorten(c *gin.Context) {
	var req UpdateShortenAuthorizedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	req.Owner = c.Param("sub")
	req.ID    = c.Param("shid")

	shorten, err := h.shortenSvc.UpdateShorten(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated my shorten successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

func (h *ShortenHandler) DeleteUserShorten(c *gin.Context) {
	userID    := c.Param("sub")
	shortenID := c.Param("shid")

	if err := h.shortenSvc.DeleteShorten(c.Request.Context(), shortenID, userID); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted user shorten successfully", nil)
}