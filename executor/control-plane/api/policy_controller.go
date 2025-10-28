package api

import (
	"errors"
	"net/http"

	"hfi/control-plane/logger"
	"hfi/control-plane/service"
	"hfi/control-plane/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PolicyController handles HTTP requests for policy operations.
type PolicyController struct {
	policyService *service.PolicyService
}

// NewPolicyController creates a new PolicyController.
func NewPolicyController(policyService *service.PolicyService) *PolicyController {
	return &PolicyController{
		policyService: policyService,
	}
}

// CreateOrUpdate handles POST /v1/policies
func (pc *PolicyController) CreateOrUpdate(c *gin.Context) {
	log := logger.WithComponent("api.policy")

	var policy storage.FaultInjectionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		log.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	policyName := policy.Metadata.Name
	policyLog := logger.WithPolicyName(policyName)

	if err := pc.policyService.CreateOrUpdatePolicy(&policy); err != nil {
		policyLog.Error("Failed to create or update policy", zap.Error(err))
		c.Error(err)
		return
	}

	policyLog.Info("Policy created or updated successfully")
	c.JSON(http.StatusCreated, policy)
}

// Create handles POST /v1/policies/create
func (pc *PolicyController) Create(c *gin.Context) {
	log := logger.WithComponent("api.policy")

	var policy storage.FaultInjectionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		log.Warn("Invalid request body for create", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	policyName := policy.Metadata.Name
	policyLog := logger.WithPolicyName(policyName)

	if err := pc.policyService.CreatePolicy(&policy); err != nil {
		policyLog.Error("Failed to create policy", zap.Error(err))
		c.Error(err)
		return
	}

	policyLog.Info("Policy created successfully")
	c.JSON(http.StatusCreated, gin.H{
		"message": "policy created successfully",
		"policy":  policy,
	})
}

// Update handles PUT /v1/policies/:id
func (pc *PolicyController) Update(c *gin.Context) {
	log := logger.WithComponent("api.policy")
	id := c.Param("id")

	if id == "" {
		log.Warn("Missing policy ID in update request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "policy id is required"})
		return
	}

	var policy storage.FaultInjectionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		log.Warn("Invalid request body for update", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Ensure the policy name in the body matches the URL parameter
	if policy.Metadata.Name != id {
		log.Warn("Policy name mismatch",
			zap.String("url_id", id),
			zap.String("body_name", policy.Metadata.Name))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "policy name in URL must match policy name in body",
		})
		return
	}

	policyLog := logger.WithPolicyName(id)

	if err := pc.policyService.UpdatePolicy(&policy); err != nil {
		policyLog.Error("Failed to update policy", zap.Error(err))

		// 显式处理不同的错误类型
		if errors.Is(err, storage.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":       "policy not found",
				"policy_name": id,
			})
		} else if errors.Is(err, service.ErrInvalidInput) || errors.Is(err, service.ErrInvalidPolicy) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid policy",
				"details": err.Error(),
			})
		} else {
			// Check for detailed error
			var detailedErr *service.DetailedError
			if errors.As(err, &detailedErr) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   detailedErr.Type,
					"message": detailedErr.Message,
					"details": detailedErr.Details,
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}
		return
	}

	policyLog.Info("Policy updated successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "policy updated successfully",
		"policy":  policy,
	})
}

// Get handles GET /v1/policies/:id
func (pc *PolicyController) Get(c *gin.Context) {
	log := logger.WithComponent("api.policy")
	id := c.Param("id")

	if id == "" {
		log.Warn("Missing policy ID in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "policy id is required"})
		return
	}

	policyLog := logger.WithPolicyName(id)
	policy, err := pc.policyService.GetPolicyByName(id)
	if err != nil {
		policyLog.Warn("Failed to get policy", zap.Error(err))
		c.Error(err)
		return
	}

	policyLog.Info("Policy retrieved successfully")
	c.JSON(http.StatusOK, policy)
}

// List handles GET /v1/policies
func (pc *PolicyController) List(c *gin.Context) {
	log := logger.WithComponent("api.policy")
	policies := pc.policyService.ListPolicies()

	log.Info("Policies listed successfully", zap.Int("count", len(policies)))
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// Delete handles DELETE /v1/policies/:id
func (pc *PolicyController) Delete(c *gin.Context) {
	log := logger.WithComponent("api.policy")
	id := c.Param("id")

	if id == "" {
		log.Warn("Missing policy ID in delete request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "policy id is required"})
		return
	}

	policyLog := logger.WithPolicyName(id)
	if err := pc.policyService.DeletePolicy(id); err != nil {
		policyLog.Warn("Failed to delete policy", zap.Error(err))
		c.Error(err)
		return
	}

	policyLog.Info("Policy deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted successfully"})
}
