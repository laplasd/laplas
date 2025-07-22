package watchdog

import (
	"context"
	"io"
	"time"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
)

const (
	/*
		RUS: Дефолтное значение периода опроса компонентов со статусом "Pending"
		ENG: Default value of the polling period for components with the "Pending" status
	*/
	DefaultPendingCheckInterval = 5 * time.Second
	/*
		RUS: Дефолтное значение периода опроса компонентов со статусом "Running"
		ENG: Default value of the polling period for components with the "Running" status
	*/
	DefaultRunningCheckInterval = 5 * time.Second
	/*
		RUS: Дефолтное значение периода опроса компонентов со статусом "Failed"
		ENG: Default value of the polling period for components with the "Failed" status
	*/
	DefaultFailedCheckInterval = 5 * time.Second
)

type WatchDog struct {
	logger               *logrus.Logger
	core                 *inforo.Core
	PendingCheckInterval *time.Duration
	RunningCheckInterval *time.Duration
	FailedCheckInterval  *time.Duration
	MaxWorkers           int
	OperationTimeout     time.Duration
}

type WatchDogOpts struct {
	Logger               *logrus.Logger
	Core                 *inforo.Core
	PendingCheckInterval *time.Duration
	RunningCheckInterval *time.Duration
	FailedCheckInterval  *time.Duration
	MaxWorkers           int
	OperationTimeout     time.Duration
}

func NewWatchDog(opts WatchDogOpts) (*WatchDog, error) {
	opts = DefaultOpts(opts)
	return &WatchDog{
		logger:               opts.Logger,
		core:                 opts.Core,
		PendingCheckInterval: opts.PendingCheckInterval,
		RunningCheckInterval: opts.RunningCheckInterval,
		FailedCheckInterval:  opts.FailedCheckInterval,
		MaxWorkers:           opts.MaxWorkers,
		OperationTimeout:     opts.OperationTimeout,
	}, nil
}

func NewDefaultWatchDog() (*WatchDog, error) {
	opts := DefaultOpts(WatchDogOpts{})

	return &WatchDog{
		logger:               opts.Logger,
		core:                 opts.Core,
		PendingCheckInterval: opts.PendingCheckInterval,
		RunningCheckInterval: opts.RunningCheckInterval,
		FailedCheckInterval:  opts.FailedCheckInterval,
		MaxWorkers:           opts.MaxWorkers,
		OperationTimeout:     opts.OperationTimeout,
	}, nil
}

func DefaultOpts(opts WatchDogOpts) WatchDogOpts {
	if opts.Logger == nil {
		opts.Logger = NewNullLogger()
	}
	if opts.Core == nil {

	}
	if opts.PendingCheckInterval == nil {
		defaultVal := DefaultPendingCheckInterval
		opts.PendingCheckInterval = &defaultVal
	}
	if opts.RunningCheckInterval == nil {
		defaultVal := DefaultRunningCheckInterval
		opts.RunningCheckInterval = &defaultVal
	}
	if opts.FailedCheckInterval == nil {
		defaultVal := DefaultFailedCheckInterval
		opts.FailedCheckInterval = &defaultVal
	}

	return opts
}

func NewNullLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = io.Discard
	return logger
}

func (wd *WatchDog) RunProcessor(ctx context.Context) {
	wd.logger.Debug("WatchDog: starting...")
	defer wd.logger.Info("WatchDog: stopped")

	go wd.RunComponentHandler(ctx)
	go wd.RunMonitoringHandler(ctx)

}

func processByStatus[T *model.Component | *model.Monitoring](
	wd *WatchDog,
	status model.Status,
	compType string,
	handler func(T),
) {
	wd.logger.Debugf("WatchDog[%s]: processing by status", status)

	// Обработка компонентов
	if compType == "components" {
		if components, err := wd.core.Components.List(); err == nil {
			for _, comp := range components {
				comp.StatusHistory.MU.RLock()
				lastStatus := comp.StatusHistory.LastStatus
				comp.StatusHistory.MU.RUnlock()

				if lastStatus != status {
					continue
				}

				if t, ok := any(comp).(T); ok {
					wd.logger.Debugf("WatchDog[%s]: component ID: %s", status, comp.ID)
					go handler(t)
				}
			}
		}
	}
	if compType == "monitorings" {
		if components, err := wd.core.Monitorings.List(); err == nil {
			for _, comp := range components {
				comp.StatusHistory.MU.RLock()
				lastStatus := comp.StatusHistory.LastStatus
				comp.StatusHistory.MU.RUnlock()

				if lastStatus != status {
					continue
				}

				if t, ok := any(comp).(T); ok {
					wd.logger.Debugf("WatchDog[%s]: component ID: %s", status, comp.ID)
					go handler(t)
				}
			}
		}
	}

}

// Обёртка, безопасная к nil-Status
func safeNextStatus(cm api.ComponentRegistry, status model.Status, current *model.StatusHistory) *model.StatusHistory {
	if current == nil {
		return cm.NewStatus(status)
	}
	return cm.NextStatus(status, current)
}
