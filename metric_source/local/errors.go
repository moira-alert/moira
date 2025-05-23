package local

import (
	"fmt"
	"strings"

	"github.com/ansel1/merry"
	"github.com/go-graphite/carbonapi/expr/helper"
)

// ErrUnknownFunction used when carbonapi.ParseExpr returns unknown function error.
type ErrUnknownFunction struct {
	FuncName      string
	internalError error
}

// ErrorUnknownFunction parses internal carbon-api error errUnknownFunction, gets func name and return ErrUnknownFunction error.
func ErrorUnknownFunction(err error) ErrUnknownFunction {
	errorStr := err.Error()
	funcName := strings.ReplaceAll(errorStr[strings.Index(errorStr, "\""):], "\"", "")

	return ErrUnknownFunction{
		internalError: err,
		FuncName:      funcName,
	}
}

// Error is implementation of golang error interface for ErrUnknownFunction struct.
func (err ErrUnknownFunction) Error() string {
	if err.FuncName == "" {
		return err.internalError.Error()
	}

	return fmt.Sprintf("Unknown graphite function: \"%s\"", err.FuncName)
}

// isErrUnknownFunction checks error for carbonapi.errUnknownFunction.
func isErrUnknownFunction(err error) bool {
	switch merry.Unwrap(err).(type) { // nolint:errorlint
	case helper.ErrUnknownFunction:
		return true
	}

	return false
}

// ErrParseExpr used when carbonapi.ParseExpr returns error.
type ErrParseExpr struct {
	internalError error
	target        string
}

// Error is implementation of golang error interface for ErrParseExpr struct.
func (err ErrParseExpr) Error() string {
	return fmt.Sprintf("failed to parse target '%s': %s", err.target, err.internalError.Error())
}

// ErrEvalExpr used when carbonapi.EvalExpr returns error.
type ErrEvalExpr struct {
	internalError error
	target        string
}

// ErrorEvalExpression creates ErrEvalExpr with given err and target.
func ErrorEvalExpression(err error, target string) ErrEvalExpr {
	return ErrEvalExpr{
		internalError: err,
		target:        target,
	}
}

// Error is implementation of golang error interface for ErrEvalExpr struct.
func (err ErrEvalExpr) Error() string {
	return fmt.Sprintf("failed to evaluate target '%s': %s", err.target, err.internalError.Error())
}

// ErrEvaluateTargetFailedWithPanic used to identify occurred error as a result of recover from panic.
type ErrEvaluateTargetFailedWithPanic struct {
	target         string
	recoverMessage interface{}
	stackRecord    []byte
}

// Error is implementation of golang error interface for ErrEvaluateTargetFailedWithPanic struct.
func (err ErrEvaluateTargetFailedWithPanic) Error() string {
	return fmt.Sprintf("panic while evaluate target %s: message: '%s' stack: %s", err.target, err.recoverMessage, err.stackRecord)
}
