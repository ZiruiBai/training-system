package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// ok writes a 200 JSON response.
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// created writes a 201 JSON response.
func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// list writes a paged list envelope.
func list(c *gin.Context, data any, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": model.PageMeta{Page: page, PageSize: pageSize, Total: total}})
}

// fail writes a structured error envelope.
func fail(c *gin.Context, status int, code, message string, details map[string]any) {
	c.JSON(status, model.ErrorEnvelope{
		Error: model.ErrorDetail{Code: code, Message: message, Details: details, RequestID: middleware.GetRequestID(c)},
	})
}

// badRequest writes a 400 error.
func badRequest(c *gin.Context, msg string) { fail(c, http.StatusBadRequest, "BAD_REQUEST", msg, nil) }

// notFound writes a 404 error.
func notFound(c *gin.Context, msg string) { fail(c, http.StatusNotFound, "NOT_FOUND", msg, nil) }

// conflict writes a 409 error.
func conflict(c *gin.Context, msg string) { fail(c, http.StatusConflict, "CONFLICT", msg, nil) }

// internal writes a generic 500 error.
func internal(c *gin.Context) {
	fail(c, http.StatusInternalServerError, "INTERNAL", "An unexpected error occurred", nil)
}

// bindJSON binds the body and returns an error response on failure.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		badRequest(c, "Invalid JSON request body")
		return false
	}
	return true
}

// parseID parses a :id path parameter.
func parseID(c *gin.Context) (uint, bool) {
	v := c.Param("id")
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil || id == 0 {
		notFound(c, "Resource not found")
		return 0, false
	}
	return uint(id), true
}

// pageParams extracts validated paging parameters.
func pageParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

// mapServiceError converts a known service error to an HTTP response.
func mapServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		badRequest(c, err.Error())
	case errors.Is(err, service.ErrNotFound):
		notFound(c, err.Error())
	case errors.Is(err, service.ErrConflict):
		conflict(c, err.Error())
	case errors.Is(err, service.ErrForbidden):
		fail(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	default:
		internal(c)
	}
}
