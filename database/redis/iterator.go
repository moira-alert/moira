package redis

import (
	"errors"
	"fmt"

	"github.com/gomodule/redigo/redis"
)

var ErrFinished = errors.New("redis iterator reached its end")

const defaultBatchSize = uint64(20)

// SetIterator scans a set, returning all it's values
// May return duplicate values
// A value may get omitted if it were not constantly present in the collection during a full iteration
type SetIterator struct {
	conn       redis.Conn
	setName    string
	dbIterator string
	batchSize  uint64

	values     []string
	currIndex  int
	isFinished bool
}

// Next returns the next iterator value
// Returns empty string and ErrFinished if there are no more values
// Returns empty string and error when fetching values from db failed
func (i *SetIterator) Next() (string, error) {
	val, err := i.nextValue()
	if err == nil {
		return val, err
	}
	if i.isFinished {
		i.Close()
		return "", ErrFinished
	}
	// making sure we skip empty responses
	for {
		i.dbIterator, i.values, err = i.receiveBatch()
		if err != nil {
			return "", fmt.Errorf("scanning the %s set failed, error: %v", i.setName, err)
		}

		i.currIndex = 0
		i.isFinished = i.dbIterator == "0"

		if len(i.values) != 0 || i.isFinished {
			break
		}
	}

	return i.nextValue()
}

// ReadToEnd iterates over the whole set and returns results
func (i *SetIterator) ReadToEnd() ([]string, error) {
	setSize, err := redis.Int(i.conn.Do("SCARD", patternsListKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get moira patterns, error: %v", err)
	}
	values := make([]string, 0, setSize)

	for {
		val, err := i.Next()
		if err == ErrFinished {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}

	return values, nil
}

// Close terminates the iterator's db connection
// Iterator closes automatically, once it is read to the end
func (i *SetIterator) Close() error {
	err := i.conn.Close()
	if err == nil || err.Error() == "redigo: closed" {
		return nil
	}
	return err
}

func (i *SetIterator) nextValue() (string, error) {
	if i.currIndex >= len(i.values) {
		return "", ErrFinished
	}
	val := i.values[i.currIndex]
	i.currIndex++
	return val, nil
}

func (i *SetIterator) receiveBatch() (next string, values []string, err error) {
	response, err := redis.Values(i.conn.Do("SSCAN", i.setName, i.dbIterator, "COUNT", i.getBatchSize()))
	if err != nil {
		return
	}
	next, err = redis.String(response[0], err)
	if err != nil {
		return
	}
	values, err = redis.Strings(response[1], err)
	if err != nil {
		return
	}
	return
}

func (i *SetIterator) getBatchSize() uint64 {
	if i.batchSize == 0 {
		return defaultBatchSize
	}
	return i.batchSize
}
