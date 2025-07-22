package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
)

type TaskHandler struct {
	logger *logrus.Logger
	mu     sync.Mutex
	core   *inforo.Core
}

func NewTaskHandler(logger *logrus.Logger, core *inforo.Core) (*TaskHandler, error) {

	taskHandler := &TaskHandler{
		logger: logger,
		core:   core,
	}

	return taskHandler, nil
}

func (t *TaskHandler) RunProcessor(ctx context.Context) {
	t.logger.Debug("TaskManager: Component Handler started")
	defer t.logger.Info("TaskManager: Component Handler stopped")

	var (
		DELAY_PENDING_STATUS = 5 * time.Second
		DELAY_RUNNING_STATUS = 5 * time.Second
		DELAY_FAILED_STATUS  = 10 * time.Second
	)

	tickers := map[model.Status]*time.Ticker{
		model.StatusPending: time.NewTicker(DELAY_PENDING_STATUS),
		model.StatusRunning: time.NewTicker(DELAY_RUNNING_STATUS),
		model.StatusFailed:  time.NewTicker(DELAY_FAILED_STATUS),
	}
	defer func() {
		for _, t := range tickers {
			t.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickers[model.StatusPending].C:
			t.processTasks(model.StatusPending, t.handlePending)
		case <-tickers[model.StatusRunning].C:
			t.processTasks(model.StatusRunning, t.recheck)
		case <-tickers[model.StatusFailed].C:
			t.processTasks(model.StatusFailed, t.retryFailed)
		}
	}
}

// Обработка компонентов по статусу с указанным обработчиком
func (t *TaskHandler) processTasks(status model.Status, handler func(*model.Task)) {
	tasks, _ := t.core.Tasks.List()
	for _, task := range tasks {
		// безопасная проверка nil статуса
		if task.StatusHistory == nil || task.StatusHistory.LastStatus != status {
			continue
		}
		// создаём копию внутри цикла для безопасной горутины
		copied := task
		go handler(copied)
	}
}

func (t *TaskHandler) handlePending(task *model.Task) {
	t.logger.Debugf("Processing pending task ID='%s'", task.ID)
	t.updateStatus(task, model.StatusCheck)
	t.addEvent(task, "Task created")
	t.checkAndUpdate(task)
}

func (t *TaskHandler) retryFailed(task *model.Task) {
	t.logger.Debugf("Retrying failed task ID='%s'", task.ID)
	if err := t.check(task); err != nil {
		t.logger.Warnf("Retry failed for task ID='%s': %v", task.ID, err)
		return
	}
	t.updateStatus(task, model.StatusRunning)
	t.addEvent(task, "Task recovered")
}

func (t *TaskHandler) recheck(task *model.Task) {
	t.logger.Debugf("Rechecking task ID='%s'", task.ID)
	if err := t.check(task); err != nil {
		t.logger.Warnf("Task ID='%s' failed recheck: %v", task.ID, err)
		t.updateStatus(task, model.StatusFailed)
		t.addEvent(task, err.Error())
		return
	}
	t.logger.Debugf("Task ID='%s' is healthy", task.ID)
}

func (t *TaskHandler) checkAndUpdate(task *model.Task) error {
	if err := t.check(task); err != nil {
		t.logger.Errorf("Check failed for task ID='%s': %v", task.ID, err)
		t.updateStatus(task, model.StatusFailed)
		t.addEvent(task, err.Error())
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.core.Tasks.Get(task.ID); err == nil {
		t.updateStatus(task, model.StatusRunning)
		t.addEvent(task, "Task started")
		t.executeTask(task)
	}
	return nil
}

func (t *TaskHandler) check(task *model.Task) error {
	t.core.Tasks.AddEvent(task.EventHistory, "Check components...")
	t.core.Tasks.Update(task.ID, task)
	/*
		component, err := t.core.Components.Get(task.ComponentID)
		if err != nil {
			return err
		}
		t.logger.Debugf("TaskManager[task:%s]: Component ID='%s'", task.ID, component.ID)

		controller, err := t.core.Controllers.Get(component.Type)
		if err != nil {
			return err
		}
		t.logger.Debugf("TaskManager[task:%s]: Controller Type='%s'", task.ID, component.Type)

		err = controller.CheckComponent(component.Metadata)
		if err != nil {
			return err
		}
	*/
	return nil
}

// Обёртка, безопасная к nil-Status
func (t *TaskHandler) safeNextStatus(cm api.ComponentRegistry, status model.Status, current *model.StatusHistory) *model.StatusHistory {
	if current == nil {
		return cm.NewStatus(status)
	}
	return cm.NextStatus(status, current)
}

func (t *TaskHandler) executeTask(task *model.Task) {
	t.logger.Infof("Execute Task '%s', status='%s'!", task.ID, task.StatusHistory.LastStatus)
	err := t.runTaskLogic(task)

	t.mu.Lock()
	defer t.mu.Unlock()
	if err != nil {
		t.logger.Errorf("Task %s failed: %v", task.ID, err)
		task.StatusHistory = t.core.Tasks.NextStatus(model.StatusFailed, task.StatusHistory)
		t.core.Tasks.Update(task.ID, task)
	} else {
		t.logger.Infof("Task %s completed successfully", task.ID)
		task.StatusHistory = t.core.Tasks.NextStatus(model.StatusSuccess, task.StatusHistory)
		t.core.Tasks.Update(task.ID, task)
	}
}

func (t *TaskHandler) runChecks(checks []*model.Check) error {
	for _, check := range checks {
		monitor, err := t.core.Monitorings.Get(check.MonitoringID)
		if err != nil {
			return err
		}
		t.logger.Infof("monitor %v for Check %s!", monitor, check.ID)
	}
	return nil
}

// stub для реальной логики выполнения задачи
func (t *TaskHandler) runTaskLogic(task *model.Task) error {

	if len(task.DependsOn) != 0 {
		t.logger.Infof("Depends for Task %s!", task.ID)

		for _, depens := range task.DependsOn {
			task, err := t.core.Tasks.Get(depens.ID)
			if err != nil {
				return err
			}
			t.logger.Infof("Start processing depends for depensID '%s'!", depens.ID)
			t.executeTask(task)
		}

	}
	/*
		component, err := t.core.Components.Get(task.ComponentID)
		if err != nil {
			return err
		}

		t.logger.Debugf("task for component %+v", component)

		controller, err := t.core.Controllers.Get(component.Type)
		if err != nil {
			return err
		}

		task.StatusHistory = t.core.Tasks.NextStatus(model.StatusRunning, *task.StatusHistory)
		t.core.Tasks.Update(task.ID, *task)
		t.logger.Infof("Running task %s (%s) for component %s of type %s", task.ID, task.Type, component.ID, component.Type)

		if task.PreChecks != nil {
			err := t.runChecks(task.PreChecks)
			if err != nil {
				return err
			}
		}

		err = controller.RunTask(task.Metadata, component.Metadata)
		if err != nil {
			task.StatusHistory = t.core.Tasks.NextStatus(model.StatusFailed, *task.StatusHistory)
			t.core.Tasks.Update(task.ID, *task)
			return err
		}
	*/
	return nil
}

func (t *TaskHandler) updateStatus(task *model.Task, status model.Status) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task.StatusHistory == nil {
		task.StatusHistory = t.core.Tasks.NewStatus(status)
	} else {
		task.StatusHistory = t.core.Tasks.NextStatus(status, task.StatusHistory)
	}
	t.core.Tasks.Update(task.ID, task)
}

func (t *TaskHandler) addEvent(task *model.Task, message string) {

	t.mu.Lock()
	defer t.mu.Unlock()

	// Получаем актуальную версию задачи
	currentTask, _ := t.core.Tasks.Get(task.ID)

	// Добавляем событие
	t.core.Tasks.AddEvent(currentTask.EventHistory, message)

	// Сохраняем обновлённую задачу
	t.core.Tasks.Update(currentTask.ID, currentTask)
}
