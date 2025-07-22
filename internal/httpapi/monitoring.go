package httpapi

import (
	"net/http"

	"github.com/laplasd/inforo/model"

	"github.com/gin-gonic/gin"
)

// POST /monitoring
func (s *APIServer) PostMonitoring(c *gin.Context) {
	var mon *model.Monitoring
	if err := c.ShouldBindJSON(&mon); err != nil {
		s.logger.Warnf("Invalid monitoring config: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"error":   "invalid monitoring config",
			"message": err,
		})
		return
	}
	monitor, err := s.core.Monitorings.Register(mon.Type, mon)
	if err != nil {
		s.logger.Warnf("Register monitoring conflict: %v", err)
		c.JSON(http.StatusConflict, gin.H{
			"code":    http.StatusConflict,
			"error":   err.Error(),
			"message": err,
		})
		return
	}
	s.logger.Infof("Monitoring system registered: %+v", mon)
	c.JSON(http.StatusCreated, gin.H{
		"code":     http.StatusCreated,
		"message":  "monitoring registered",
		"metadata": monitor})
}

// GET /monitoring/:id
func (s *APIServer) GetMonitoring(c *gin.Context) {
	id := c.Param("id")
	mon, err := s.core.Monitorings.Get(id)
	if err != nil {
		s.logger.Warnf("Monitoring %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "monitoring not found"})
		return
	}
	c.JSON(http.StatusOK, mon)
}

// PUT /monitoring/:id
func (s *APIServer) UpdateMonitoring(c *gin.Context) {
	id := c.Param("id")

	var updated *model.Monitoring
	if err := c.ShouldBindJSON(&updated); err != nil {
		s.logger.Warnf("Invalid monitoring data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitoring config"})
		return
	}

	if err := s.core.Monitorings.Update(id, updated); err != nil {
		s.logger.Warnf("Failed to update monitoring %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	s.logger.Infof("Monitoring %s updated", id)
	c.Status(http.StatusOK)
}

// DELETE /monitoring/:id
func (s *APIServer) DeleteMonitoring(c *gin.Context) {
	id := c.Param("id")

	if err := s.core.Monitorings.Delete(id); err != nil {
		s.logger.Warnf("Failed to delete monitoring %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	s.logger.Infof("Monitoring %s deleted", id)
	c.Status(http.StatusOK)
}

// GET /monitoring
func (s *APIServer) ListMonitoring(c *gin.Context) {
	all, _ := s.core.Monitorings.List()
	c.JSON(http.StatusOK, all)
}
