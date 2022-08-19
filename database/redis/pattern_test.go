package redis

import (
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUpdatePatternList(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "info", "test", true)
	dataBase := NewTestDatabase(logger)
	defer dataBase.Flush()

	pattern := "my-super-pattern"

	Convey("Successful update", t, func() {
		defer dataBase.Flush()

		patterns, err := dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(patterns, ShouldBeEmpty)

		pipe := (*dataBase.client).TxPipeline()
		pipe.SAdd(dataBase.context, patternTriggersKey(pattern), "my-trigger-id")
		_, err = pipe.Exec(dataBase.context)
		So(err, ShouldBeNil)

		err = dataBase.UpdatePatternList()
		So(err, ShouldBeNil)

		patterns, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(patterns, ShouldNotBeEmpty)
		So(patterns, ShouldResemble, []string{"my-super-pattern"})
	})

	Convey("Nothing to update", t, func() {
		defer dataBase.Flush()

		patterns, err := dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(patterns, ShouldBeEmpty)

		pipe := (*dataBase.client).TxPipeline()
		pipe.SAdd(dataBase.context, patternTriggersKey(pattern), "my-trigger-id")
		pipe.SAdd(dataBase.context, patternsListKey, pattern)
		_, err = pipe.Exec(dataBase.context)
		So(err, ShouldBeNil)

		err = dataBase.UpdatePatternList()
		So(err, ShouldBeNil)

		patterns, err = dataBase.GetPatterns()
		So(err, ShouldBeNil)
		So(patterns, ShouldNotBeEmpty)
		So(patterns, ShouldResemble, []string{"my-super-pattern"})
	})
}
