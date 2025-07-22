package httpapi

import (
	"net/http"

	"github.com/laplasd/inforo/model"

	"github.com/gin-gonic/gin"
)

// POST /components
func (s *APIServer) CreateComponent(c *gin.Context) {
	var comp *model.Component
	if err := c.ShouldBindJSON(&comp); err != nil {
		s.logger.Warnf("APIServer.CreateComponent: Invalid component: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid component"})
		return
	}
	comp, err := s.core.Components.Register(*comp)

	if err != nil {
		s.logger.Warnf("APIServer.CreateComponent: Register component conflict: %v", err)
		c.JSON(http.StatusConflict, gin.H{
			"code":  http.StatusConflict,
			"error": err.Error(),
		})
		return
	}
	s.logger.Infof("APIServer.CreateComponent: Component registered: %+v", comp)
	c.JSON(http.StatusCreated, gin.H{
		"code":     http.StatusCreated,
		"message":  "component registered successfully",
		"metadata": comp,
	})
}

// GET /components/:id
func (s *APIServer) GetComponent(c *gin.Context) {
	id := c.Param("id")
	comp, err := s.core.Components.Get(id)
	if err != nil {
		s.logger.Warnf("Component %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"error":   "component not found",
			"message": err,
		})
		return
	}
	c.JSON(http.StatusOK, comp)
}

// PUT /components/:id
func (s *APIServer) UpdateComponent(c *gin.Context) {
	id := c.Param("id")

	var updatedComp model.Component
	if err := c.ShouldBindJSON(&updatedComp); err != nil {
		s.logger.Warnf("Invalid component data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"error":   "invalid component",
			"message": err.Error(),
		})
		return
	}

	err := s.core.Components.Update(id, &updatedComp)

	if err != nil {
		s.logger.Warnf("Failed to update component %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	s.logger.Infof("Component %s updated", id)
	component, _ := s.core.Components.Get(id)
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"message":  "Component updated",
		"metadata": component,
	})
}

// DELETE /components/:id
func (s *APIServer) DeleteComponent(c *gin.Context) {
	id := c.Param("id")

	if err := s.core.Components.Delete(id); err != nil {
		s.logger.Warnf("Failed to delete component %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	s.logger.Infof("Component %s deleted", id)
	c.Status(http.StatusOK)
}

// GET /components
func (s *APIServer) ListComponents(c *gin.Context) {
	filterType := c.Query("type")
	all, _ := s.core.Components.List()

	// Если фильтра нет — возвращаем всё
	if filterType == "" {
		c.JSON(http.StatusOK, all)
		return
	}

	// Фильтрация по типу
	var filtered []*model.Component
	for _, comp := range all {
		if comp.Type == filterType {
			filtered = append(filtered, comp)
		}
	}

	c.JSON(http.StatusOK, filtered)
}

func (s *APIServer) DisableComponent(c *gin.Context) {
	id := c.Param("id")
	err := s.core.Components.Disable(id)
	if err != nil {
		s.logger.Warnf("Component %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"error":   "component not found",
			"message": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Component disabled",
	})
}

func (s *APIServer) EnableComponent(c *gin.Context) {
	id := c.Param("id")
	err := s.core.Components.Enable(id)
	if err != nil {
		s.logger.Warnf("Component %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"error":   "component not found",
			"message": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Component Enabled",
	})
}
