package watchdog

import (
	"context"
	"time"

	"github.com/laplasd/inforo/model"
)

func (wd *WatchDog) RunComponentHandler(ctx context.Context) {
	wd.logger.Debug("WatchDog: Component Handler starting...")
	defer wd.logger.Info("WatchDog: stopped")

	var pendingChan <-chan time.Time // Объявляем канал, а не тикер
	if *wd.PendingCheckInterval != 0 {
		pendingTicker := time.NewTicker(*wd.PendingCheckInterval)
		wd.logger.Debugf("WatchDog: PendingCheckInterval: %s", wd.PendingCheckInterval)
		defer pendingTicker.Stop()
		pendingChan = pendingTicker.C // Используем только канал
	}

	var runningChan <-chan time.Time // Объявляем канал, а не тикер
	if *wd.RunningCheckInterval != 0 {
		runningTicker := time.NewTicker(*wd.RunningCheckInterval)
		wd.logger.Debugf("WatchDog: RunningCheckInterval: %s", wd.RunningCheckInterval)
		defer runningTicker.Stop()
		runningChan = runningTicker.C // Используем только канал
	}

	var failedChan <-chan time.Time // Объявляем канал, а не тикер
	if *wd.FailedCheckInterval != 0 {
		failedTicker := time.NewTicker(*wd.FailedCheckInterval)
		wd.logger.Debugf("WatchDog: FailedCheckInterval: %s", wd.FailedCheckInterval)
		defer failedTicker.Stop()
		failedChan = failedTicker.C // Используем только канал
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-pendingChan:
			processByStatus[*model.Component](wd, model.StatusPending, "components", wd.pendingComponent)
		case <-runningChan:
			processByStatus[*model.Component](wd, model.StatusRunning, "components", wd.recheck)
		case <-failedChan:
			processByStatus[*model.Component](wd, model.StatusFailed, "components", wd.retryFailed)
		}
	}
}

func (wd *WatchDog) pendingComponent(comp *model.Component) {

	comp.StatusHistory = safeNextStatus(wd.core.Components, model.StatusCheck, comp.StatusHistory)
	err := wd.core.Components.Update(comp.ID, comp)
	if err != nil {
		wd.core.Components.AddEvent(comp.EventHistory, err.Error())
		wd.logger.Errorf("WatchDog[pending]: Failed to update component ID='%s': %v", comp.ID, err)
		return
	}

	wd.checkAndUpdate(comp)
}

func (c *WatchDog) retryFailed(comp *model.Component) {
	c.logger.Debugf("Retrying failed component ID='%s'", comp.ID)

	err := c.check(comp)
	if err != nil {
		c.logger.Warnf("Retry failed for component ID='%s': %v", comp.ID, err)
		return
	}

	//comp.MU.Lock()
	if _, err := c.core.Components.Get(comp.ID); err == nil {
		comp.StatusHistory = c.core.Components.NextStatus(model.StatusRunning, comp.StatusHistory)
		c.core.Components.AddEvent(comp.EventHistory, "Component recovered and set to RUNNING")
		c.core.Components.Update(comp.ID, comp)
		c.logger.Infof("Component ID='%s' recovered and set to RUNNING", comp.ID)
	}
	//comp.MU.Unlock()
}

func (c *WatchDog) recheck(comp *model.Component) {

	checkeErr := c.check(comp)
	if checkeErr != nil {
		c.logger.Warnf("Component ID='%s' failed recheck: %v", comp.ID, checkeErr)
		if _, err := c.core.Components.Get(comp.ID); err == nil {
			comp.StatusHistory = safeNextStatus(c.core.Components, model.StatusFailed, comp.StatusHistory)
			c.core.Components.AddEvent(comp.EventHistory, checkeErr.Error())
			c.core.Components.Update(comp.ID, comp)
		}
		return
	}

	c.logger.Debugf("WatchDog[running]: Component '%s' is still healthy", comp.ID)
}

func (c *WatchDog) checkAndUpdate(comp *model.Component) error {
	c.logger.Debugf("Checking component ID='%s'", comp.ID)

	err := c.check(comp)
	if err != nil {
		c.logger.Errorf("Check failed for component ID='%s': %v", comp.ID, err)
		comp.StatusHistory = c.core.Components.NextStatus(model.StatusFailed, comp.StatusHistory)
		c.core.Components.AddEvent(comp.EventHistory, err.Error())
		err := c.core.Components.Update(comp.ID, comp)
		if err != nil {
			return err
		}

		return nil
	}

	//c.mu.Lock()
	//defer c.mu.Unlock()

	// Обновляем статус компонента в карте, если он всё ещё есть
	if _, err := c.core.Components.Get(comp.ID); err == nil {
		comp.StatusHistory = c.core.Components.NextStatus(model.StatusRunning, comp.StatusHistory)
		c.core.Components.AddEvent(comp.EventHistory, "Component status updated to RUNNING")
		c.core.Components.Update(comp.ID, comp)
		c.logger.Infof("Component ID='%s' status updated to RUNNING", comp.ID)
	}
	return nil
}

func (c *WatchDog) check(component *model.Component) error {

	controller, err := c.core.Controllers.Get(component.Type)
	if err != nil {
		return err
	}

	err = controller.CheckComponent(component.Metadata)
	if err != nil {
		return err
	}
	return nil
}
