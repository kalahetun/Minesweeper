package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"hfi/control-plane/storage"
)

// ErrorHandlerMiddleware handles errors that are attached to the gin context.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Check for specific error types
			if errors.Is(err, storage.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "resource not found",
					"details": err.Error(),
				})
				return
			}

			// For all other errors, return 500 Internal Server Error
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
				"details": err.Error(),
			})
		}
	}
}
