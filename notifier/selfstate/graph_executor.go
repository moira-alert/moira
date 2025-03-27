package selfstate

import (
	"errors"
	"sync"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/selfstate/heartbeat"
)

type graphExecutionResult struct {
	currentValue int64
	hasErrors bool
	needTurnOffNotifier bool
	errorMessages []string
	checksTags []string
}

type heartbeaterCheckResult struct {
	currentValue int64
	hasErrors bool
	error error
	needTurnOffNotifier bool
	errorMessage string
	checkTags []string
}

func ExecuteGraph(graph [][]heartbeat.Heartbeater, nowTS int64) (graphExecutionResult, error) {
	var wg sync.WaitGroup
	for _, layer := range graph {
		layerResult, err := runHeartbeatersLayer(layer, nowTS, &wg)
		if layerResult.hasErrors || err != nil {
			return layerResult, err
		}
	}
	return graphExecutionResult{
		currentValue: 0,
		hasErrors: false,
		needTurnOffNotifier: false,
		errorMessages: nil,
		checksTags: nil,
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
	arr := make([]heartbeaterCheckResult, len(results))
	for r := range results {
		arr = append(arr, r)
	}
	merged, err := mergeLayerResults(arr...)
	return merged, err
}

func runHeartbeaterCheck(heartbeater heartbeat.Heartbeater, nowTS int64, wg *sync.WaitGroup, resultChan chan<- heartbeaterCheckResult) {
	currentValue, hasErrors, err := heartbeater.Check(nowTS)

	var needTurnOffNotifier bool
	var errorMessage string
	var checkTags []string

	if hasErrors {
		needTurnOffNotifier = heartbeater.NeedTurnOffNotifier()
		errorMessage = heartbeater.GetErrorMessage()
		checkTags = heartbeater.GetCheckTags()
	}
	resultChan <- heartbeaterCheckResult{
		currentValue: currentValue,
		hasErrors: hasErrors,
		needTurnOffNotifier: needTurnOffNotifier,
		errorMessage: errorMessage,
		checkTags: checkTags,
		error: err,
	}
	wg.Done()
}

func mergeLayerResults(results ...heartbeaterCheckResult) (graphExecutionResult, error) {
	var result graphExecutionResult
	for _, res := range results {
		if res.hasErrors {
			result.hasErrors = result.hasErrors || res.hasErrors
			result.currentValue = moira.MaxInt64(result.currentValue, res.currentValue)
			result.errorMessages = append(result.errorMessages, res.errorMessage)
			result.needTurnOffNotifier = result.needTurnOffNotifier || res.needTurnOffNotifier
		}
		result.checksTags = append(result.checksTags, res.checkTags...)
	}
	errs := errors.Join(moira.Map(results, func(r heartbeaterCheckResult) error { return r.error })...)

	return result, errs
}

