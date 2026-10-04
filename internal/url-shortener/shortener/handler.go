package shortener

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
		pkg.ErrorResponse(c, http.StatusNotFound, pkg.ErrNotFound)
	case errors.Is(err, pkg.ErrForbidden):
		pkg.ErrorResponse(c, http.StatusForbidden, pkg.ErrForbidden)
	case errors.Is(err, pkg.ErrInvalidInput):
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
	default:
		pkg.ErrorResponse(c, http.StatusInternalServerError, pkg.ErrInternal)
	}
}

// Shorten Public

// @Summary      Redirect to Original URL
// @Description  Redirect to Original URL from Shorten Code in Redis Cache
// @Tags         URL Shortener Public
// @Produce      json
// @Param        code path string true "Shorten Code from Generate Shorten Link"
// @Success      302 {object} pkg.Response "Redirecting to Original URL"
// @Failure      404 {object} pkg.Response "Shorten Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /s/{code} [get]
func (h *ShortenHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("code")

	originalURL, err := h.shortenSvc.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		httpError(c, err)
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

// @Summary      Create Shorten Code Public
// @Description  Create Shorten Code from OriginalURL
// @Tags         URL Shortener Public
// @Accept       json
// @Produce      json
// @Param        request body CreateShortenPublicRequest true "Create Shorten Public Request"
// @Success      201 {object} ShortenerSwaggerResponse "Created Short URL Successfully"
// @Failure      401 {object} pkg.Response "Invalid Body Requesy"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /s [post]
func (h *ShortenHandler) CreateShorten(c *gin.Context) {
	var req CreateShortenPublicRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shorten, err := h.shortenSvc.CreateShortenPublic(c.Request.Context(), req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "create shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// Shorten Authorized

// @Summary      Create Shorten Code Authorized
// @Description  Create Shorten Code from OriginalURL
// @Tags         URL Shortener Authorized
// @Accept       json
// @Produce      json
// @Param        request body CreateShortenAuthorizedRequest true "Create Shorten Authorized Request"
// @Success      201 {object} ShortenerSwaggerResponse "Created Short URL Successfully"
// @Failure      400 {object} pkg.Response "Invalid Body Requesy"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me [post]
func (h *ShortenHandler) CreateMyShorten(c *gin.Context) {
	var req CreateShortenAuthorizedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.Owner = c.GetString("userID")

	shorten, err := h.shortenSvc.CreateShortenAuthorized(c.Request.Context(), &req)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, "create shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Get All Shortens Code Paginated List
// @Description  Get All Shortens Code Data from database with Paginated List
// @Tags         URL Shortener Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        page  query int false "Page (Default: 1)"
// @Param        limit query int false "Limit (Default: 10)"
// @Success      200 {object} ShortensListSwaggerResponse "Get Shortens Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me [get]
func (h *ShortenHandler) GetMyShortens(c *gin.Context) {
	ownerID := c.GetString("userID")
	page, limit, err := pkg.ParsePagination(c, 10, 100)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shortens, err := h.shortenSvc.GetShortenByOwner(c.Request.Context(), ownerID, page, limit)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get shortens code successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

// @Summary      Get Shortener Code by Shorten ID(shid)
// @Description  Get Shortener Code Data from database by Shorten ID(shid)
// @Tags         URL Shortener Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        shid path string true "Shorten ID"
// @Success      200 {object} ShortenerSwaggerResponse "Get Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me/{shid} [get]
func (h *ShortenHandler) GetMyShortenByID(c *gin.Context) {
	shortenID := c.Param("shid")
	userID := c.GetString("userID")
	role := c.GetString("role")

	shorten, err := h.shortenSvc.GetShortenByID(c.Request.Context(), shortenID, userID, role)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Update Shortener Code by Shorten ID(shid)
// @Description  Update Shortener Code Data from database by Shorten ID(shid)
// @Tags         URL Shortener Authorized
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        shid path string true "Shorten ID"
// @Param        request body UpdateShortenAuthorizedRequest true "Update Shortener Authorized Request"
// @Success      200 {object} ShortenerSwaggerResponse "Updated Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me/{shid} [patch]
func (h *ShortenHandler) UpdateMyShorten(c *gin.Context) {
	var req UpdateShortenAuthorizedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.ID = c.Param("shid")
	req.Owner = c.GetString("userID")
	role := c.GetString("role")

	shorten, err := h.shortenSvc.UpdateShorten(c.Request.Context(), &req, role)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Update Status Shortener Code by Shorten ID(shid) with Query Status(active or inactive)
// @Description  Update Status Shortener Code Data from database by Shorten ID(shid) with Query Status(active or inactive)
// @Tags         URL Shortener Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        shid   path  string true "Shorten ID"
// @Param        status query string true "Status: active | inactive"
// @Success      200 {object} ShortenerSwaggerResponse "Updated Status Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me/{shid} [put]
func (h *ShortenHandler) SetMyShortenStatus(c *gin.Context) {
	shid := c.Param("shid")
	owner := c.GetString("userID")
	role := c.GetString("role")
	status := c.DefaultQuery("status", "active")

	if status != "active" && status != "inactive" {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shorten, err := h.shortenSvc.SetURLStatus(c.Request.Context(), owner, shid, role, status)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated status shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Delete Shortener Code by Shorten ID(shid)
// @Description  Delete Shortener Code Data from database by Shorten ID(shid)
// @Tags         URL Shortener Authorized
// @Security     BearerAuth
// @Produce      json
// @Param        shid path  string true "Shorten ID"
// @Success      200 {object} pkg.Response "Deleted Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /shortens/me/{shid} [delete]
func (h *ShortenHandler) DeleteMyShorten(c *gin.Context) {
	shortenID := c.Param("shid")
	userID := c.GetString("userID")
	role := c.GetString("role")

	if err := h.shortenSvc.DeleteShorten(c.Request.Context(), shortenID, userID, role); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted shortener code successfully", nil)
}

// @Summary      Get All User Shortens Code Paginated List (Admin)
// @Description  Get All User Shortens Code Data from database with Paginated List
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        page  query int false "Page (Default: 1)"
// @Param        limit query int false "Limit (Default: 10)"
// @Success      200 {object} ShortensListSwaggerResponse "Get Shortens Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/shortens [get]
func (h *ShortenHandler) GetUserShortens(c *gin.Context) {
	page, limit, err := pkg.ParsePagination(c, 10, 100)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shortens, err := h.shortenSvc.GetShortens(c.Request.Context(), page, limit)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get shortens code successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

// @Summary      Get User Shortener Code by Shorten ID(shid) (Admin)
// @Description  Get Shortener Code Data from database by Shorten ID(shid)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub  path string true "User ID"
// @Param        shid path string true "Shorten ID"
// @Success      200 {object} ShortenerSwaggerResponse "Get Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/shortens/{shid} [get]
func (h *ShortenHandler) GetUserShortenByID(c *gin.Context) {
	userID := c.Param("sub")
	shortenID := c.Param("shid")
	role := c.GetString("role")

	shorten, err := h.shortenSvc.GetShortenByID(c.Request.Context(), shortenID, userID, role)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Get User Shortens Code Paginated List by Shorten ID(shid) (Admin)
// @Description  Get User Shortens Code Paginated List from database by Shorten ID(shid)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        sub   path  string true  "User ID"
// @Param        page  query int    false "Page (Default: 1)"
// @Param        limit query int    false "Limit (Default: 10)"
// @Success      200 {object} ShortenerSwaggerResponse "Get Shortens Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/shortens [get]
func (h *ShortenHandler) GetShortensByUserID(c *gin.Context) {
	ownerID := c.Param("sub")
	page, limit, err := pkg.ParsePagination(c, 10, 100)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shortens, err := h.shortenSvc.GetShortenByOwner(c.Request.Context(), ownerID, page, limit)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "get shortens code successfully", map[string]interface{}{
		"shortens": shortens,
	})
}

// @Summary      Update User Shortener Code by Shorten ID(shid)
// @Description  Update User Shortener Code Data from database by Shorten ID(shid)
// @Tags         Admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        sub  path string true "User ID"
// @Param        shid path string true "Shorten ID"
// @Param        request body UpdateShortenAuthorizedRequest true "Update Shortener Authorized Request"
// @Success      200 {object} ShortenerSwaggerResponse "Updated Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/shortens/{shid} [patch]
func (h *ShortenHandler) UpdateUserShorten(c *gin.Context) {
	var req UpdateShortenAuthorizedRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	req.Owner = c.Param("sub")
	req.ID = c.Param("shid")
	role := c.GetString("role")

	shorten, err := h.shortenSvc.UpdateShorten(c.Request.Context(), &req, role)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Update Status Shortener Code by Shorten ID(shid) with Query Status(active or inactive) (Admin)
// @Description  Update Status Shortener Code Data from database by Shorten ID(shid) with Query Status(active or inactive)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        shid   path  string true "Shorten ID"
// @Param        sub    path  string true "User ID"
// @Param        status query string true "Status: active | inactive"
// @Success      200 {object} ShortenerSwaggerResponse "Updated Status Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/shortens/{shid} [put]
func (h *ShortenHandler) SetUserShortenStatus(c *gin.Context) {
	owner := c.Param("sub")
	shid := c.Param("shid")
	role := c.GetString("role")
	status := c.DefaultQuery("status", "active")

	if status != "active" && status != "inactive" {
		pkg.ErrorResponse(c, http.StatusBadRequest, pkg.ErrInvalidInput)
		return
	}

	shorten, err := h.shortenSvc.SetURLStatus(c.Request.Context(), owner, shid, role, status)
	if err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "updated status shortener code successfully", map[string]interface{}{
		"shorten": shorten,
	})
}

// @Summary      Delete Shortener Code by Shorten ID(shid) (Admin)
// @Description  Delete Shortener Code Data from database by Shorten ID(shid)
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Param        shid   path  string true "Shorten ID"
// @Param        sub    path  string true "User ID"
// @Success      200 {object} ShortenerSwaggerResponse "Delete Shortener Code Successfully"
// @Failure      401 {object} pkg.Response "Unauthorized"
// @Failure      403 {object} pkg.Response "Access Denied"
// @Failure      404 {object} pkg.Response "Code Not Found"
// @Failure      500 {object} pkg.Response "Internal Server Error"
// @Router       /admin/users/{sub}/shortens/{shid} [delete]
func (h *ShortenHandler) DeleteUserShorten(c *gin.Context) {
	userID := c.Param("sub")
	shortenID := c.Param("shid")
	role := c.GetString("role")

	if err := h.shortenSvc.DeleteShorten(c.Request.Context(), shortenID, userID, role); err != nil {
		httpError(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, "deleted shortener code successfully", nil)
}
