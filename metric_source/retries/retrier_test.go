package retries

import (
	"errors"
	"github.com/cenkalti/backoff/v4"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStandardRetrier(t *testing.T) {
	var (
		maxRetriesCount    uint64 = 6
		testErr                   = errors.New("some test err")
		errInsidePermanent        = errors.New("test err inside permanent")
		permanentTestErr          = backoff.Permanent(errInsidePermanent)
	)

	conf := Config{
		InitialInterval:     testInitialInterval,
		RandomizationFactor: testRandomizationFactor,
		Multiplier:          testMultiplier,
		MaxInterval:         testMaxInterval,
		MaxRetriesCount:     maxRetriesCount,
	}

	retrier := NewStandardRetrier[int]()

	Convey("Test retrier", t, func() {
		Convey("with successful RetryableOperation", func() {
			retPairs := []retPair[int]{
				{
					returnValue: 25,
					err:         nil,
				},
				{
					returnValue: 26,
					err:         nil,
				},
			}
			expectedCalls := 1

			stub := newStubRetryableOperation[int](retPairs)

			backoffPolicy := NewExponentialBackoffFactory(conf).NewBackOff()

			gotRes, gotErr := retrier.Retry(stub, backoffPolicy)

			So(gotRes, ShouldEqual, retPairs[0].returnValue)
			So(gotErr, ShouldBeNil)
			So(stub.calls, ShouldEqual, expectedCalls)
		})

		Convey("with successful RetryableOperation after some retries", func() {
			retPairs := []retPair[int]{
				{
					returnValue: 25,
					err:         testErr,
				},
				{
					returnValue: 10,
					err:         testErr,
				},
				{
					returnValue: 42,
					err:         nil,
				},
				{
					returnValue: 41,
					err:         nil,
				},
			}
			expectedCalls := 3

			stub := newStubRetryableOperation[int](retPairs)

			backoffPolicy := NewExponentialBackoffFactory(conf).NewBackOff()

			gotRes, gotErr := retrier.Retry(stub, backoffPolicy)

			So(gotRes, ShouldEqual, retPairs[2].returnValue)
			So(gotErr, ShouldBeNil)
			So(stub.calls, ShouldEqual, expectedCalls)
		})

		Convey("with permanent error from RetryableOperation after some retries", func() {
			retPairs := []retPair[int]{
				{
					returnValue: 25,
					err:         testErr,
				},
				{
					returnValue: 10,
					err:         permanentTestErr,
				},
				{
					returnValue: 42,
					err:         nil,
				},
				{
					returnValue: 41,
					err:         nil,
				},
			}
			expectedCalls := 2

			stub := newStubRetryableOperation[int](retPairs)

			backoffPolicy := NewExponentialBackoffFactory(conf).NewBackOff()

			gotRes, gotErr := retrier.Retry(stub, backoffPolicy)

			So(gotRes, ShouldEqual, retPairs[1].returnValue)
			So(gotErr, ShouldResemble, errInsidePermanent)
			So(stub.calls, ShouldEqual, expectedCalls)
		})

		Convey("with RetryableOperation failed on each retry", func() {
			expectedCalls := conf.MaxRetriesCount + 1

			stub := newStubRetryableOperation[int](nil)

			backoffPolicy := NewExponentialBackoffFactory(conf).NewBackOff()

			gotRes, gotErr := retrier.Retry(stub, backoffPolicy)

			So(gotRes, ShouldEqual, 0)
			So(gotErr, ShouldResemble, errStubValuesEnded)
			So(stub.calls, ShouldEqual, expectedCalls)
		})
	})
}

type retPair[T any] struct {
	returnValue T
	err         error
}

type stubRetryableOperation[T any] struct {
	retPairs []retPair[T]
	idx      int
	calls    int
}

func newStubRetryableOperation[T any](pairs []retPair[T]) *stubRetryableOperation[T] {
	return &stubRetryableOperation[T]{
		retPairs: pairs,
		idx:      0,
		calls:    0,
	}
}

var (
	errStubValuesEnded = errors.New("prepared return values and errors for stub ended")
)

func (stub *stubRetryableOperation[T]) DoRetryableOperation() (T, error) {
	stub.calls += 1

	if stub.idx >= len(stub.retPairs) {
		return *new(T), errStubValuesEnded
	}

	res := stub.retPairs[stub.idx]
	stub.idx += 1

	return res.returnValue, res.err
}
