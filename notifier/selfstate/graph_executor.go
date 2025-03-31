package selfstate

import (
	"errors"
	"sync"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

type graphExecutionResult struct {
	currentValue        int64
	hasErrors           bool
	needTurnOffNotifier bool
	errorMessages       []string
	checksTags          []string
}

type heartbeaterCheckResult struct {
	lastSuccessCheckElapsedTime int64
	hasErrors                   bool
	error                       error
	needTurnOffNotifier         bool
	errorMessage                string
	checkTags                   []string
}

// executeGraph executes a series of heartbeater checks in a layered graph structure.
func (graph heartbeatsGraph) executeGraph(nowTS int64) (graphExecutionResult, error) {
	var wg sync.WaitGroup
	for _, layer := range graph {
		layerResult, err := runHeartbeatersLayer(layer, nowTS, &wg)
		if layerResult.hasErrors || err != nil {
			return layerResult, err
		}
	}

	return graphExecutionResult{
		currentValue:        0,
		hasErrors:           false,
		needTurnOffNotifier: false,
		errorMessages:       nil,
		checksTags:          nil,
	}, nil
}

func runHeartbeatersLayer(graphLayer []heartbeat.Heartbeater, nowTS int64, wg *sync.WaitGroup) (graphExecutionResult, error) {
	results := make(chan heartbeaterCheckResult, len(graphLayer))
	for _, heartbeat := range graphLayer {
		wg.Add(1)
		go runHeartbeaterCheck(heartbeat, nowTS, wg, results)
	}

	wg.Wait()
	close(results)
	arr := make([]heartbeaterCheckResult, 0, len(results))

	for r := range results {
		arr = append(arr, r)
	}
	merged, err := mergeLayerResults(arr...)

	return merged, err
}

func runHeartbeaterCheck(heartbeater heartbeat.Heartbeater, nowTS int64, wg *sync.WaitGroup, resultChan chan<- heartbeaterCheckResult) {
	lastSuccessCheckElapsedTime, hasErrors, err := heartbeater.Check(nowTS)

	var needTurnOffNotifier bool
	var errorMessage string
	var checkTags []string

	if hasErrors {
		needTurnOffNotifier = heartbeater.NeedTurnOffNotifier()
		errorMessage = heartbeater.GetErrorMessage()
		checkTags = heartbeater.GetCheckTags()
	}

	resultChan <- heartbeaterCheckResult{
		lastSuccessCheckElapsedTime: lastSuccessCheckElapsedTime,
		hasErrors:                   hasErrors,
		needTurnOffNotifier:         needTurnOffNotifier,
		errorMessage:                errorMessage,
		checkTags:                   checkTags,
		error:                       err,
	}

	wg.Done()
}

func mergeLayerResults(layersResults ...heartbeaterCheckResult) (graphExecutionResult, error) {
	var graphResult graphExecutionResult
	for _, layerResult := range layersResults {
		if layerResult.hasErrors {
			graphResult.hasErrors = graphResult.hasErrors || layerResult.hasErrors
			graphResult.currentValue = moira.MaxInt64(graphResult.currentValue, layerResult.lastSuccessCheckElapsedTime)
			graphResult.errorMessages = append(graphResult.errorMessages, layerResult.errorMessage)
			graphResult.needTurnOffNotifier = graphResult.needTurnOffNotifier || layerResult.needTurnOffNotifier
		}

		graphResult.checksTags = append(graphResult.checksTags, layerResult.checkTags...)
	}

	errs := errors.Join(moira.Map(layersResults, func(r heartbeaterCheckResult) error { return r.error })...)

	return graphResult, errs
}
