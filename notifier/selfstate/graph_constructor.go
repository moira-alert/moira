package selfstate

import "github.com/moira-alert/moira/notifier/selfstate/heartbeat"

type heartbeatsGraph [][]heartbeat.Heartbeater

// constructHeartbeatsGraph constructs a graph of heartbeats based on their order and blocking.
func constructHeartbeatsGraph(heartbeats []heartbeat.Heartbeater) heartbeatsGraph {
	var graph heartbeatsGraph
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
