package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /components
func (s *APIServer) ListControllers(c *gin.Context) {
	filterType := c.Query("type")
	all, _ := s.core.Controllers.ListType()

	// Если фильтра нет — возвращаем всё
	if filterType == "" {
		c.JSON(http.StatusOK, all)
		return
	}

	// Фильтрация по типу
	var filtered []string
	for _, comp := range all {
		if comp == filterType {
			filtered = append(filtered, comp)
		}
	}

	c.JSON(http.StatusOK, filtered)
}
