package target

import (
	"fmt"
	"strings"
)

// ErrUnknownFunction used when carbonapi.ParseExpr returns unknown function error
type ErrUnknownFunction struct {
	InternalError error
}

// Error is implementation of golang error interface for ErrUnknownFunction struct
func (err ErrUnknownFunction) Error() string {
	return err.InternalError.Error()
}

// isErrUnknownFunction checks error for carbonapi.errUnknownFunction
func isErrUnknownFunction(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown function in evalExpr")
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
