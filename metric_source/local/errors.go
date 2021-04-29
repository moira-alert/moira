package local

import (
	"fmt"
	"strings"

	"github.com/go-graphite/carbonapi/expr/helper"
)

// ErrUnknownFunction used when carbonapi.ParseExpr returns unknown function error
type ErrUnknownFunction struct {
	FuncName      string
	internalError error
}

// ErrorUnknownFunction parses internal carbon-api error errUnknownFunction, gets func name and return ErrUnknownFunction error
func ErrorUnknownFunction(err error) ErrUnknownFunction {
	errorStr := err.Error()
	funcName := strings.Replace(errorStr[strings.Index(errorStr, "\""):], "\"", "", -1)
	return ErrUnknownFunction{
		internalError: err,
		FuncName:      funcName,
	}
}

// Error is implementation of golang error interface for ErrUnknownFunction struct
func (err ErrUnknownFunction) Error() string {
	if err.FuncName == "" {
		return err.internalError.Error()
	}
	return fmt.Sprintf("Unknown graphite function: \"%s\"", err.FuncName)
}

// isErrUnknownFunction checks error for carbonapi.errUnknownFunction
func isErrUnknownFunction(err error) bool {
	switch err.(type) {
	case helper.ErrUnknownFunction:
		return true
	}
	return false
}

// ErrParseExpr used when carbonapi.ParseExpr returns error
type ErrParseExpr struct {
	internalError error
	target        string
}

// Error is implementation of golang error interface for ErrParseExpr struct
func (err ErrParseExpr) Error() string {
	return fmt.Sprintf("failed to parse target '%s': %s", err.target, err.internalError.Error())
}

// ErrEvalExpr used when carbonapi.EvalExpr returns error
type ErrEvalExpr struct {
	internalError error
	target        string
}

// Error is implementation of golang error interface for ErrEvalExpr struct
func (err ErrEvalExpr) Error() string {
	return fmt.Sprintf("failed to evaluate target '%s': %s", err.target, err.internalError.Error())
}

// ErrEvaluateTargetFailedWithPanic used to identify occurred error as a result of recover from panic
type ErrEvaluateTargetFailedWithPanic struct {
	target         string
	recoverMessage interface{}
	stackRecord    []byte
}

// Error is implementation of golang error interface for ErrEvaluateTargetFailedWithPanic struct
func (err ErrEvaluateTargetFailedWithPanic) Error() string {
	return fmt.Sprintf("panic while evaluate target %s: message: '%s' stack: %s", err.target, err.recoverMessage, err.stackRecord)
}

// errDifferentPatternsTimeRangesBuilder is a builder pattern implementation for Error different patterns time ranges
type errDifferentPatternsTimeRangesBuilder struct {
	result      *ErrDifferentPatternsTimeRanges
	returnError bool
}

// newErrDifferentPatternsTimeRangesBuilder is a constructor function for errDifferentPatternsTimeRangesBuilder
func newErrDifferentPatternsTimeRangesBuilder() errDifferentPatternsTimeRangesBuilder {
	return errDifferentPatternsTimeRangesBuilder{
		result:      &ErrDifferentPatternsTimeRanges{},
		returnError: false,
	}
}

// addPatterns is a method that adds a patterns with time ranges to error
func (b *errDifferentPatternsTimeRangesBuilder) addPattern(pattern string, from, until int64) {
	b.returnError = true
	b.result.patterns = append(b.result.patterns, fmt.Sprintf("%s: from: %d, until: %d", pattern, from, until))
}

// addCommon is a function that add to error initial time ranges with which we will compare patterns all paterns
func (b *errDifferentPatternsTimeRangesBuilder) addCommon(pattern string, from, until int64) {
	b.result.patterns = append(b.result.patterns, fmt.Sprintf("%s: from: %d, until: %d", pattern, from, until))
}

// build is a function that returns error if there exists patterns with inconsistency in time ranges
func (b *errDifferentPatternsTimeRangesBuilder) build() error {
	if b.returnError {
		return *(b.result)
	}
	return nil
}

// ErrDifferentPatternsTimeRanges is a type that represents error for situation in which one target have couple patterns
// and this patterns have different time ranges. That behavior can appear if some patterns have aggregation functions or
// scale function with different time arguments
type ErrDifferentPatternsTimeRanges struct {
	patterns []string
}

// Error is an error interface implementation method
func (err ErrDifferentPatternsTimeRanges) Error() string {
	return fmt.Sprintf("Some of patterns have different time ranges in the same target:\n%s", strings.Join(err.patterns, "\n"))
}
