package watchdog

import (
	"context"
	"time"

	"github.com/laplasd/inforo/model"
)

func (wd *WatchDog) RunMonitoringHandler(ctx context.Context) {
	wd.logger.Debug("Monitor: Monitoring Handler started")
	defer wd.logger.Info("Monitor: Handler stopped")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	runningTicker := time.NewTicker(5 * time.Second)
	defer runningTicker.Stop()

	failedTicker := time.NewTicker(10 * time.Second)
	defer failedTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processByStatus[*model.Monitoring](wd, model.StatusPending, "monitorings", wd.pendingMonitor)
		case <-runningTicker.C:
			processByStatus[*model.Monitoring](wd, model.StatusRunning, "monitorings", wd.recheckMonitor)
		case <-failedTicker.C:
			processByStatus[*model.Monitoring](wd, model.StatusFailed, "monitorings", wd.retryFailedMonitor)
		}
	}
}

func (wd *WatchDog) pendingMonitor(comp *model.Monitoring) {
	wd.logger.Debugf("ComponentManager: Pending status ID='%s'", comp.ID)

	comp.StatusHistory = wd.core.Monitorings.NextStatus(model.StatusRunning, comp.StatusHistory)
	err := wd.core.Monitorings.Update(comp.ID, comp)
	if err != nil {
		wd.core.Monitorings.AddEvent(comp.EventHistory, err.Error())
		wd.logger.Errorf("Failed to update component ID='%s': %v", comp.ID, err)
		return
	}

	wd.checkAndUpdateMonitor(comp)
}

func (wd *WatchDog) retryFailedMonitor(comp *model.Monitoring) {
	wd.logger.Debugf("Retrying failed component ID='%s'", comp.ID)

	err := wd.checkMonitor(comp)
	if err != nil {
		wd.logger.Warnf("Retry failed for component ID='%s': %v", comp.ID, err)
		return
	}

	if _, err := wd.core.Monitorings.Get(comp.ID); err == nil {
		comp.StatusHistory = wd.core.Monitorings.NextStatus(model.StatusRunning, comp.StatusHistory)
		wd.core.Monitorings.AddEvent(comp.EventHistory, "Component recovered and set to RUNNING")
		wd.core.Monitorings.Update(comp.ID, comp)
		wd.logger.Infof("Component ID='%s' recovered and set to RUNNING", comp.ID)
	}
}

func (wd *WatchDog) recheckMonitor(comp *model.Monitoring) {
	wd.logger.Debugf("Rechecking component ID='%s'", comp.ID)

	checkeErr := wd.checkMonitor(comp)
	if checkeErr != nil {
		wd.logger.Warnf("Component ID='%s' failed recheck: %v", comp.ID, checkeErr)
		if _, err := wd.core.Monitorings.Get(comp.ID); err == nil {
			comp.StatusHistory = wd.core.Monitorings.NextStatus(model.StatusRunning, comp.StatusHistory)
			wd.core.Monitorings.AddEvent(comp.EventHistory, checkeErr.Error())
			wd.core.Monitorings.Update(comp.ID, comp)
		}
		return
	}

	wd.logger.Debugf("Component ID='%s' is still healthy", comp.ID)
}

func (wd *WatchDog) checkAndUpdateMonitor(comp *model.Monitoring) error {
	wd.logger.Debugf("Checking component ID='%s'", comp.ID)

	err := wd.checkMonitor(comp)
	if err != nil {
		wd.logger.Errorf("Check failed for component ID='%s': %v", comp.ID, err)
		comp.StatusHistory = wd.core.Monitorings.NextStatus(model.StatusFailed, comp.StatusHistory)
		wd.core.Monitorings.AddEvent(comp.EventHistory, err.Error())
		err := wd.core.Monitorings.Update(comp.ID, comp)
		if err != nil {
			return err
		}

		return nil
	}

	// Обновляем статус компонента в карте, если он всё ещё есть
	if _, err := wd.core.Monitorings.Get(comp.ID); err == nil {
		comp.StatusHistory = wd.core.Monitorings.NextStatus(model.StatusRunning, comp.StatusHistory)
		wd.core.Monitorings.AddEvent(comp.EventHistory, "Component status updated to RUNNING")
		wd.core.Monitorings.Update(comp.ID, comp)
		wd.logger.Infof("Component ID='%s' status updated to RUNNING", comp.ID)
	}
	return nil
}

func (wd *WatchDog) checkMonitor(monitor *model.Monitoring) error {

	controller, err := wd.core.MonitorControllers.Get(monitor.Type)
	if err != nil {
		return err
	}

	err = controller.CheckMonitoring(monitor.Config)
	if err != nil {
		return err
	}
	return nil
}
