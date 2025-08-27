package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"hfi/control-plane/logger"
	"hfi/control-plane/service"
	"hfi/control-plane/storage"
)

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Code    string                 `json:"code,omitempty"`
}

// ErrorHandlerMiddleware handles errors that are attached to the gin context.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			log := logger.WithComponent("api.error_handler")

			// Log the error for debugging
			log.Error("Request processing error", 
				zap.Error(err),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
			)

			// Handle specific error types
			var statusCode int
			var errorResponse ErrorResponse

			switch {
			case errors.Is(err, storage.ErrNotFound):
				statusCode = http.StatusNotFound
				errorResponse = ErrorResponse{
					Error:   "not_found",
					Message: "The requested resource was not found",
					Details: map[string]interface{}{
						"resource": "policy",
					},
				}

			case errors.Is(err, storage.ErrAlreadyExists):
				statusCode = http.StatusConflict
				errorResponse = ErrorResponse{
					Error:   "resource_already_exists",
					Message: "The resource already exists",
					Details: map[string]interface{}{
						"resource": "policy",
					},
				}

			case errors.Is(err, storage.ErrInvalidInput):
				statusCode = http.StatusBadRequest
				errorResponse = ErrorResponse{
					Error:   "invalid_input",
					Message: "The provided input is invalid",
				}

			case errors.Is(err, service.ErrInvalidInput):
				statusCode = http.StatusBadRequest
				errorResponse = ErrorResponse{
					Error:   "invalid_input",
					Message: "The provided input is invalid",
				}

				// Check if it's a detailed error with more context
				var detailedErr *service.DetailedError
				if errors.As(err, &detailedErr) {
					errorResponse.Message = detailedErr.Message
					errorResponse.Details = detailedErr.Details
					errorResponse.Code = detailedErr.Type
				}

			case errors.Is(err, service.ErrInvalidPolicy):
				statusCode = http.StatusBadRequest
				errorResponse = ErrorResponse{
					Error:   "invalid_policy",
					Message: "The policy configuration is invalid",
				}

			case errors.Is(err, service.ErrPolicyExists):
				statusCode = http.StatusConflict
				errorResponse = ErrorResponse{
					Error:   "policy_already_exists",
					Message: "A policy with this name already exists",
				}

			default:
				// Check for validation errors
				var validationErr *service.ValidationError
				if errors.As(err, &validationErr) {
					statusCode = http.StatusBadRequest
					errorResponse = ErrorResponse{
						Error:   "validation_error",
						Message: "Input validation failed",
						Details: map[string]interface{}{
							"field":           validationErr.Field,
							"validation_message": validationErr.Message,
						},
					}
				} else {
					// For all other errors, return 500 Internal Server Error
					statusCode = http.StatusInternalServerError
					errorResponse = ErrorResponse{
						Error:   "internal_server_error",
						Message: "An internal server error occurred",
						Details: map[string]interface{}{
							"error_type": "unknown",
						},
					}

					// Log additional details for internal server errors
					log.Error("Unhandled error type", 
						zap.String("error_type", "unknown"),
						zap.String("error_message", err.Error()),
					)
				}
			}

			c.JSON(statusCode, errorResponse)
		}
	}
}
