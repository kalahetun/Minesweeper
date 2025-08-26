package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"hfi/control-plane/logger"
	"hfi/control-plane/service"
	"hfi/control-plane/storage"
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
