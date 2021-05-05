package redis

import (
	"errors"

	"github.com/gomodule/redigo/redis"
)

// DbCursor implements DB cursor abstraction
type DbCursor struct {
	client     *DbConnector
	connection redis.Conn
	nextCursor int
	queryArgs  []interface{}
	endReached bool
}

// NewCursor returns cursor from 'SCAN 0 ' with adjusted args (MATCH, COUNT, TYPE, ...)
func (connector *DbConnector) NewCursor(args ...interface{}) DbCursor {
	return DbCursor{
		client:     connector,
		nextCursor: 0,
		queryArgs:  args,
		endReached: false,
	}
}

func (c *DbCursor) Next() ([]interface{}, error) {
	if c.endReached {
		return nil, errors.New("the end of collection was reached")
	}
	if c.connection == nil {
		c.connection = c.client.pool.Get()
	}
	args := append([]interface{}{c.nextCursor}, c.queryArgs...)
	r, err := c.connection.Do("SCAN", args...)
	var updatedCursor int
	var list []interface{}
	values, err := redis.Values(r, err)
	if err != nil {
		return nil, err
	}
	scan, err := redis.Scan(values, &updatedCursor, &list)
	if err != nil {
		return scan, err
	}
	c.nextCursor = updatedCursor
	if updatedCursor == 0 {
		c.endReached = true
		_ = c.Free()
	}
	return list, err
}

func (c *DbCursor) Free() error {
	return c.connection.Close()
	// todo: maybe tomb?
}
