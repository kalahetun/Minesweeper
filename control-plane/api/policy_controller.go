package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	var policy storage.FaultInjectionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if err := pc.policyService.CreateOrUpdatePolicy(&policy); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// Get handles GET /v1/policies/:id
func (pc *PolicyController) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "policy id is required"})
		return
	}

	policy, err := pc.policyService.GetPolicyByName(id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// List handles GET /v1/policies
func (pc *PolicyController) List(c *gin.Context) {
	policies := pc.policyService.ListPolicies()
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// Delete handles DELETE /v1/policies/:id
func (pc *PolicyController) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "policy id is required"})
		return
	}

	if err := pc.policyService.DeletePolicy(id); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "policy deleted successfully"})
}
