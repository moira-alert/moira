package selfstate

import "github.com/moira-alert/moira/notifier/selfstate/heartbeat"

type HeartbeatsGraph [][]heartbeat.Heartbeater

func ConstructHeartbeatsGraph(heartbeats []heartbeat.Heartbeater) [][]heartbeat.Heartbeater {
	var graph [][]heartbeat.Heartbeater
	var currentLayer []heartbeat.Heartbeater

	for _, hb := range heartbeats {
		if !hb.NeedToCheckOthers() {
			graph = append(graph, []heartbeat.Heartbeater{hb})
		} else {
			currentLayer = append(currentLayer, hb)
		}
	}

	if len(currentLayer) > 0 {
		graph = append(graph, currentLayer)
	}

	return graph
}

