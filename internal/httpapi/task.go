package httpapi

import (
	"net/http"

	"github.com/laplasd/inforo/model"

	"github.com/gin-gonic/gin"
)

// POST /tasks
func (s *APIServer) CreateTask(c *gin.Context) {
	var task *model.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		s.logger.Warnf("Invalid task data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":     http.StatusBadRequest,
			"message":  "invalid task",
			"metadata": err,
		})
		return
	}

	fullTask, err := s.core.Tasks.Register(task)

	if err != nil {
		s.logger.Warnf("Failed to create task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":     http.StatusInternalServerError,
			"message":  "Failed to create task",
			"metadata": err.Error(),
		})
		return
	}
	s.logger.Infof("Task created: %+v", task)
	c.JSON(http.StatusCreated, gin.H{
		"code":     http.StatusCreated,
		"message":  "Task created",
		"metadata": fullTask,
	})
}

// GET /tasks
func (s *APIServer) ListTasks(c *gin.Context) {
	tasks, err := s.core.Tasks.List()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":     http.StatusInternalServerError,
			"message":  "Internal Server Error",
			"metadata": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"message":  "ListTasks",
		"metadata": tasks,
	})
}

// GET /tasks/:id
func (s *APIServer) GetTask(c *gin.Context) {
	id := c.Param("id")
	task, err := s.core.Tasks.Get(id)
	if err != nil {
		s.logger.Warnf("Task %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// PUT /tasks/:id
func (s *APIServer) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var updatedTask *model.Task
	if err := c.ShouldBindJSON(&updatedTask); err != nil {
		s.logger.Warnf("Invalid task data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task"})
		return
	}

	if err := s.core.Tasks.Update(id, updatedTask); err != nil {
		s.logger.Warnf("Failed to update task %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	s.logger.Infof("Task %s updated", id)
	c.JSON(http.StatusOK, updatedTask)
}

// DELETE /tasks/:id
func (s *APIServer) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := s.core.Tasks.Delete(id); err != nil {
		s.logger.Warnf("Failed to delete task %s: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	s.logger.Infof("Task %s deleted", id)
	c.Status(http.StatusOK)
}

// GET /tasks/:id
func (s *APIServer) RunTask(c *gin.Context) {
	id := c.Param("id")
	procID, err := s.core.Tasks.ForkAsync(id, "")
	if err != nil {
		s.logger.Warnf("Task %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, procID)
}

func (s *APIServer) RollBackTask(c *gin.Context) {
	id := c.Param("id")
	procID, err := s.core.Tasks.RollBackAsync(id, "")
	if err != nil {
		s.logger.Warnf("Task %s not found: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, procID)
}
