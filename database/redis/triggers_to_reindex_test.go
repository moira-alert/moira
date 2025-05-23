package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggersToReindex(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabase(logger)
	dataBase.Flush()

	defer dataBase.Flush()

	Convey("Test on empty DB", t, func() {
		actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix())
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		err = dataBase.RemoveTriggersToReindex(time.Now().Unix())
		So(err, ShouldBeNil)
	})

	Convey("Trigger to update add and fetch", t, func() {
		triggerID1 := uuid.Must(uuid.NewV4()).String()
		triggerID2 := uuid.Must(uuid.NewV4()).String()
		triggerID3 := uuid.Must(uuid.NewV4()).String()

		actual, err := dataBase.FetchTriggersToReindex(time.Now().Unix())
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		startTime := time.Now().Unix()

		// current time ≈ startTime + 1
		time.Sleep(time.Second)

		err = addTriggersToReindex(dataBase, triggerID1)
		So(err, ShouldBeNil)

		// current time ≈ startTime + 2
		time.Sleep(time.Second)
		actual, err = dataBase.FetchTriggersToReindex(time.Now().Unix())
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		// current time ≈ startTime + 3
		time.Sleep(time.Second)

		err = addTriggersToReindex(dataBase, triggerID2, triggerID3)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)

		err = dataBase.RemoveTriggersToReindex(startTime + 2)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 2)

		err = dataBase.RemoveTriggersToReindex(startTime + 4)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		// try to save 2 similar triggers
		err = addTriggersToReindex(dataBase, triggerID1, triggerID1)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		// and again
		// current time ≈ startTime + 4
		time.Sleep(time.Second)

		err = addTriggersToReindex(dataBase, triggerID1, triggerID1)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		// and now try to remove on time before last changes, nothing should change
		err = dataBase.RemoveTriggersToReindex(startTime - 10)
		So(err, ShouldBeNil)

		actual, err = dataBase.FetchTriggersToReindex(startTime)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, []string{triggerID1})

		// add other triggers several times
		err = addTriggersToReindex(dataBase, triggerID1, triggerID1)
		So(err, ShouldBeNil)

		err = addTriggersToReindex(dataBase, triggerID3, triggerID2, triggerID1)
		So(err, ShouldBeNil)

		err = addTriggersToReindex(dataBase, triggerID3, triggerID3, triggerID3, triggerID1, triggerID2)
		So(err, ShouldBeNil)

		// it's startTime + 4 now, so should return 3 triggers
		actual, err = dataBase.FetchTriggersToReindex(startTime + 3)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)
	})
}

func TestTriggerToReindexConnection(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()

	defer dataBase.Flush()

	Convey("Should throw error when no connection", t, func() {
		triggerID, err := dataBase.FetchTriggersToReindex(time.Now().Unix())
		So(triggerID, ShouldBeEmpty)
		So(err, ShouldNotBeNil)

		err = dataBase.RemoveTriggersToReindex(time.Now().Unix())
		So(err, ShouldNotBeNil)
	})
}

func addTriggersToReindex(connector *DbConnector, triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}

	ctx := connector.context
	pipe := (*connector.client).TxPipeline()

	for _, triggerID := range triggerIDs {
		z := &redis.Z{Score: float64(time.Now().Unix()), Member: triggerID}
		pipe.ZAdd(ctx, triggersToReindexKey, z)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add triggers to reindex: %s", err.Error())
	}

	return nil
}
