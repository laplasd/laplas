package httpapi

import (
	"net/http"

	"github.com/laplasd/inforo/model"

	"github.com/gin-gonic/gin"
)

// POST /plans
func (s *APIServer) CreatePlan(c *gin.Context) {
	var tasks []*model.Task

	if err := c.ShouldBindJSON(&tasks); err != nil {
		s.logger.Warnf("Invalid plan payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":     http.StatusBadRequest,
			"message":  "invalid plan format",
			"metadata": err.Error(),
		})
		return
	}

	plan, err := s.core.Plans.Register(tasks)
	if err != nil {
		s.logger.Errorf("Failed to create plan: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":     http.StatusInternalServerError,
			"message":  "Failed to create plan",
			"metadata": err.Error()})
		return
	}

	s.logger.Infof("Plan created: %s", plan.ID)
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"message":  "Plan created!",
		"metadata": plan,
	})
}

// GET /plans/:id/status
func (s *APIServer) GetPlanStatus(c *gin.Context) {
	id := c.Param("id")

	plan, err := s.core.Plans.Get(id)
	if err != nil {
		s.logger.Warnf("Plan %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    string(plan.StatusHistory.LastStatus),
		"timestamp": plan.StatusHistory.Timestamp,
		"history":   plan.StatusHistory.Previous,
	})
}

// GET /plans
func (s *APIServer) ListPlans(c *gin.Context) {
	plans, _ := s.core.Plans.List()

	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
	})
}

// GET /plans/:id
func (s *APIServer) GetPlan(c *gin.Context) {
	id := c.Param("id")

	plan, err := s.core.Plans.Get(id)
	if err != nil {
		s.logger.Warnf("Plan %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// DELETE /plans/:id
func (s *APIServer) DeletePlan(c *gin.Context) {
	id := c.Param("id")

	err := s.core.Plans.Delete(id)
	if err != nil {
		s.logger.Warnf("Failed to delete plan %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	s.logger.Infof("Plan %s deleted", id)
	c.Status(http.StatusNoContent)
}

func (s *APIServer) RunPlan(c *gin.Context) {
	id := c.Param("id")
	procID, err := s.core.Plans.RunAsync(id, "")
	if err != nil {
		s.logger.Warnf("Task %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, procID)
}
