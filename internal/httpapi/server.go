package httpapi

import (
	"fmt"
	"laplasd/internal/config"
	"net"
	"os"

	"github.com/laplasd/inforo"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type APIServer struct {
	core     *inforo.Core
	logger   *logrus.Logger
	router   *gin.Engine
	sockPath string
	IP       string
	Port     int
}

func New(core *inforo.Core, sockPath string, logger *logrus.Logger, cfg config.Server) *APIServer {

	gin.SetMode(gin.ReleaseMode) // чтобы не выводить дебаг-логи Gin по умолчанию
	router := gin.New()

	// Добавляем кастомный логгер и middleware для логов
	router.Use(gin.LoggerWithWriter(logger.Writer()))
	router.Use(gin.Recovery())

	s := &APIServer{
		core:     core,
		sockPath: cfg.UnixSocket,
		IP:       cfg.Host,
		Port:     cfg.Port,
		logger:   logger,
		router:   router,
	}

	s.setupRoutes()

	return s
}

func (s *APIServer) setupRoutes() {

	/*
		/component* Handlers
	*/
	component := s.router.Group("/component")
	{
		component.POST("", s.CreateComponent)
		component.GET("/:id", s.GetComponent)
		component.PATCH("/:id", s.UpdateComponent)
		component.DELETE("/:id", s.DeleteComponent)
		component.POST("/disable/:id", s.DisableComponent)
		component.POST("/enable/:id", s.EnableComponent)
	}

	/*
		/components* Handlers
	*/
	components := s.router.Group("/components")
	{
		components.GET("", s.ListComponents)
	}

	/*
		/monitoring* Handlers
	*/
	monitoring := s.router.Group("/monitoring")
	{
		monitoring.POST("", s.PostMonitoring)
		monitoring.GET("/:id", s.GetMonitoring)
		monitoring.PUT("/:id", s.UpdateMonitoring)
		monitoring.DELETE("/:id", s.DeleteMonitoring)
	}

	/*
		/monitorings* Handlers
	*/
	monitorings := s.router.Group("/monitorings")
	{
		monitorings.GET("", s.ListMonitoring)
	}

	/*
		/task* Handlers
	*/
	task := s.router.Group("/task")
	{
		task.POST("", s.CreateTask)
		task.GET("/:id", s.GetTask)
		task.PUT("/:id", s.UpdateTask)
		task.DELETE("/:id", s.DeleteTask)
		task.POST("/run/:id", s.RunTask)
		task.POST("/rollback/:id", s.RollBackTask)
	}

	/*
		/tasks* Handlers
	*/
	tasks := s.router.Group("/tasks")
	{
		tasks.GET("", s.ListTasks)
	}

	/*
		/plan* Handlers
	*/
	plan := s.router.Group("/plan")
	{
		plan.POST("", s.CreatePlan)
		plan.GET("/:id", s.GetPlan)
		plan.DELETE("/:id", s.DeletePlan)
		plan.GET("/:id/status", s.GetPlanStatus)
		plan.POST("/run/:id", s.RunPlan)
	}
	s.router.GET("/plans", s.ListPlans)

	//s.router.POST("/plans/:id/run", s.handleRunPlan)

	// contollers
	s.router.GET("/controllers", s.ListControllers)

}

func (s *APIServer) Start() error {
	// Удаляем unix сокет, если существует
	if err := os.RemoveAll(s.sockPath); err != nil {
		return err
	}

	unixListener, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return err
	}

	if os.Getenv("LAPLAS_ENABLE_TCP") == "1" {
		tcpListener, err := net.Listen("tcp", "127.0.0.1:8080")
		if err != nil {
			return fmt.Errorf("failed to start TCP listener: %w", err)
		}
		s.logger.Infof("API available on TCP 127.0.0.1:8080")

		go func() {
			if err := s.router.RunListener(tcpListener); err != nil {
				s.logger.Errorf("TCP server error: %v", err)
			}
		}()
	}

	s.logger.Infof("API listening on %s", s.sockPath)
	return s.router.RunListener(unixListener)
}
