package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
)

// ErrorHandler is middleware for handling application errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			if appErr, ok := errors.IsAppError(err); ok {
				c.JSON(appErr.HTTPCode, gin.H{
					"success": false,
					"error": gin.H{
						"code":    appErr.Code,
						"message": appErr.Message,
						"details": appErr.Details,
					},
				})
				return
			}

			// Handle other error types
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    errors.ErrInternalServer,
					"message": "Internal server error",
				},
			})
		}
	}
}
