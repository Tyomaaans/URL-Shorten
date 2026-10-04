package pkg

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

var errInvalidPagination = errors.New("invalid pagination")

func ParsePagination(c *gin.Context, defaultLimit, maxLimit int) (int, int, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		return 0, 0, errInvalidPagination
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, 0, errInvalidPagination
	}

	return page, limit, nil
}
