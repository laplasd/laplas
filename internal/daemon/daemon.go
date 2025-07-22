package daemon

import (
	"context"
	"laplasd/internal/config"
	"laplasd/internal/controllers"
	"laplasd/internal/handlers/watchdog"
	"laplasd/internal/httpapi"
	"laplasd/internal/logger"
	"os"

	"github.com/laplasd/inforo"

	"github.com/sirupsen/logrus"
)

var (
	unixSocket string = "/tmp/laplasd.sock"
)

type Daemon struct {
	logger  *logrus.Logger
	core    *inforo.Core
	config  *config.Config
	running bool
}

func New(logger *logrus.Logger, cfg *config.Config) *Daemon {
	return &Daemon{
		logger: logger,
		config: cfg,
	}
}

func (d *Daemon) Run() error {
	d.running = true
	d.logger.Info("Daemon: starting...")

	// Инициализация core
	err := d.initCore()
	if err != nil {
		return err
	}

	d.core.MonitorControllers.Register("promql-monitor", controllers.NewPromQLMonitorController(d.logger, "http://prometheus:9090/api/v1"))

	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = d.initHandlers(ctx)
	if err != nil {
		return err
	}

	// Удаляем старый сокет, если он существует
	if err := os.Remove(unixSocket); err != nil && !os.IsNotExist(err) {
		d.logger.Errorf("Failed to remove socket file: %v", err)
		return err
	}

	// Инициализация и запуск API (один раз)
	api := httpapi.New(d.core, unixSocket, logger.Log, d.config.Server)
	go func() {
		if err := api.Start(); err != nil {
			d.logger.Fatalf("API error: %v", err)
		}
	}()

	// Основной цикл просто проверяет флаг running
	for d.running {
		// Можно добавить небольшую паузу, чтобы не нагружать CPU
		// time.Sleep(100 * time.Millisecond)
	}

	d.logger.Info("Daemon: stopped")
	return nil
}

func (d *Daemon) Stop() {
	d.running = false
	d.logger.Info("Daemon: stopping")
}

func (d *Daemon) initCore() error {

	d.logger.Debugf("Daemon: Init Core")

	opts := inforo.CoreOptions{
		Logger: d.logger,
	}
	d.logger.Debugf("Daemon: Init Core with opts: %v", opts)
	d.core = inforo.NewCore(opts)
	return nil
}

func (d *Daemon) initHandlers(ctx context.Context) error {
	d.logger.Debugf("Daemon: Init Handlers")

	/*
		Watchdog - Обработчик и очень важный
	*/

	watchDogOpts := watchdog.WatchDogOpts{
		Logger:               d.logger,
		Core:                 d.core,
		PendingCheckInterval: d.config.WatchDog.PendingCheckInterval,
		RunningCheckInterval: d.config.WatchDog.RunningCheckInterval,
		FailedCheckInterval:  d.config.WatchDog.FailedCheckInterval,
		MaxWorkers:           d.config.WatchDog.MaxWorkers,
		OperationTimeout:     d.config.WatchDog.OperationTimeout,
	}
	watchdog, err := watchdog.NewWatchDog(watchDogOpts)
	if err != nil {
		return err
	}

	/*
		Monitor - Обработчик и очень важный
	*/
	/*
		monitorOpts := monitor.MonitorOpts{
			Logger: d.logger,
			Core:   d.core,
		}
		monitoringHandler, err := monitor.NewMonitor(monitorOpts)
		if err != nil {
			return err
		}


			taskHandler, err := handlers.NewTaskHandler(d.logger, core)
			if err != nil {
				return err
			}
	*/

	// Запускаем обработчики в горутинах (один раз)
	go watchdog.RunProcessor(ctx)
	//go monitoringHandler.RunProcessor(ctx)
	//go taskHandler.RunProcessor(ctx)

	return nil
}
