package redis

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

const setName = "test-set"

func TestSetIterator(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	db := newTestDatabase(logger, config)
	db.flush()
	defer db.flush()

	Convey("Set iteration", t, func() {
		db.clearSet()

		Convey("Empty set", func() {
			iter := db.createIterator()

			val, err := iter.Next()

			So(val, ShouldBeEmpty)
			So(err, ShouldEqual, ErrFinished)
		})

		Convey("Single item", func() {
			item := "foo"
			db.addToSet(item)
			iter := db.createIterator()

			val, err := iter.Next()
			So(err, ShouldBeNil)
			So(val, ShouldEqual, item)

			_, err = iter.Next()
			So(err, ShouldEqual, ErrFinished)
		})

		Convey("Multiple items, iterate via Next", func() {
			db.fillSetWithAlphabet()
			iter := db.createIterator()
			values := []string{}

			for {
				val, err := iter.Next()
				if err != nil {
					So(err, ShouldEqual, ErrFinished)
					break
				}
				values = append(values, val)
			}

			So(values, ShouldHaveLength, 26) //alphabet
		})

		Convey("Multiple items, read to end", func() {
			db.fillSetWithAlphabet()
			iter := db.createIterator()

			values, err := iter.ReadToEnd()

			So(err, ShouldBeNil)
			So(values, ShouldHaveLength, 26) //alphabet
		})

		Convey("Multiple items, read to end with preset batch size", func() {
			db.fillSetWithAlphabet()
			iter := db.createIterator()
			iter.batchSize = 60

			values, err := iter.ReadToEnd()

			So(err, ShouldBeNil)
			So(values, ShouldHaveLength, 26) //alphabet
		})

		Convey("Close returns nil error, after iterator is closed", func() {
			iter := db.createIterator()

			_, err := iter.Next()
			So(err, ShouldEqual, ErrFinished)

			err = iter.Close()
			So(err, ShouldBeNil)
		})
	})
}

func (c *DbConnector) clearSet() {
	conn := c.pool.Get()
	values, err := redis.Strings(conn.Do("SMEMBERS", setName))
	if err != nil {
		panic(err)
	}
	for _, val := range values {
		conn.Send("SREM", setName, val)
	}
	conn.Flush()
}

func (c *DbConnector) createIterator() SetIterator {
	return SetIterator{
		conn:       c.pool.Get(),
		setName:    setName,
		dbIterator: "0",
	}
}

func (c *DbConnector) addToSet(value string) {
	conn := c.pool.Get()
	_, err := redis.Int64(conn.Do("SADD", setName, value))
	if err != nil {
		panic(err)
	}
}

func (c *DbConnector) fillSetWithAlphabet() {
	for i := int32('A'); i <= int32('Z'); i++ {
		c.addToSet(string(rune(i)))
	}
}
