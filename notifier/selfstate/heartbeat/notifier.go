package heartbeat

import (
	"fmt"

	"github.com/moira-alert/moira"
)

type notifier struct {
	heartbeat
	clusterKey moira.ClusterKey
}

// GetNotifier returns new Heartbeater that checks notifier state for given cluster key.
func GetNotifier(defaultTags []string, tagPrefix string, localTag []string, clusterKey moira.ClusterKey, logger moira.Logger, database moira.Database) Heartbeater {
	tags := MakeNotifierTags(defaultTags, tagPrefix, localTag, clusterKey)

	return &notifier{
		heartbeat: heartbeat{
			database:  database,
			logger:    logger,
			checkTags: tags,
		},
		clusterKey: clusterKey,
	}
}

// MakeNotifierTags returns list of tags for a notifier Heartbeater with given configuration.
func MakeNotifierTags(defaultTags []string, tagPrefix string, localTags []string, clusterKey moira.ClusterKey) []string {
	tags := make([]string, 0)
	tags = append(tags, defaultTags...)
	tags = append(tags, tagPrefix+":"+clusterKey.String())

	if clusterKey == moira.DefaultLocalCluster {
		tags = append(tags, localTags...)
	}

	return tags
}

func (check notifier) Check(int64) (int64, bool, error) {
	state, _ := check.database.GetNotifierState()
	if state.State != moira.SelfStateOK && state.Actor != moira.SelfStateActorAutomatic {
		check.logger.Error().
			String("error", check.GetErrorMessage()).
			Msg("Notifier is not healthy")

		return 0, true, nil
	}

	state, _ = check.database.GetNotifierStateForSource(check.clusterKey)
	if state.State != moira.SelfStateOK && state.Actor != moira.SelfStateActorAutomatic {
		check.logger.Error().
			String("metric_source", check.clusterKey.String()).
			String("error", check.GetErrorMessage()).
			Msg("Notifier is not healthy")

		return 0, true, nil
	}

	check.logger.Debug().
		String("state", state.State).
		Msg("Notifier is healthy")

	return 0, false, nil
}

func (notifier) NeedTurnOffNotifier() bool {
	return false
}

func (notifier) NeedToCheckOthers() bool {
	return true
}

func (check notifier) GetErrorMessage() string {
	state, _ := check.database.GetNotifierState()
	if state.State == moira.SelfStateOK {
		state, _ = check.database.GetNotifierStateForSource(check.clusterKey)
	}

	return fmt.Sprintf("Moira-Notifier does not send messages. State: %v", state.State)
}

func (check *notifier) GetCheckTags() CheckTags {
	return check.checkTags
}
