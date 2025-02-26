package webhook

import (
	"time"

	"github.com/moira-alert/moira/worker"
)

const (
	webhookDeliveryCheckLockKeyPrefix = "moira-webhook-delivery-check-lock:"
	webhookDeliveryCheckLockTTL       = 30 * time.Second
	workerName                        = "WebhookDeliveryChecker"
)

func webhookLockKey(contactType string) string {
	return webhookDeliveryCheckLockKeyPrefix + contactType
}

func (sender *Sender) runDeliveryCheckWorker(contactType string) {
	workerAction := func(stop <-chan struct{}) error {
		sender.bot.Start()
		<-stop
		sender.bot.Stop()
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.log,
		sender.DataBase.NewLock(webhookLockKey(contactType), webhookDeliveryCheckLockTTL),
		workerAction,
	).Run(nil)
}
