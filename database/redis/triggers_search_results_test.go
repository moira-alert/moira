package redis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/rs/zerolog"
	. "github.com/smartystreets/goconvey/convey"
)

var searchResults = []*moira.SearchResult{
	{
		ObjectID: "TestTrigger1",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test",
			},
		},
	},
	{
		ObjectID: "TestTrigger2",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test2",
			},
		},
	},
	{
		ObjectID: "TestTrigger3",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test3",
			},
		},
	},
	{
		ObjectID: "TestTrigger4",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test4",
			},
		},
	},
	{
		ObjectID: "TestTrigger5",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test5",
			},
		},
	},
	{
		ObjectID: "TestTrigger6",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test6",
			},
		},
	},
	{
		ObjectID: "TestTrigger7",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test7",
			},
		},
	},
	{
		ObjectID: "TestTrigger8",
		Highlights: []moira.SearchHighlight{
			{
				Field: "Name",
				Value: "Test8",
			},
		},
	},
}

const searchResultsID = "Test"

func TestTriggersSearchResultsStoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	Convey("Search Results Manipulation", t, func() {
		err := dataBase.SaveTriggersSearchResults(searchResultsID, searchResults)
		So(err, ShouldBeNil)

		actual, total, err := dataBase.GetTriggersSearchResults(searchResultsID, 0, -1)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, searchResults)
		So(total, ShouldResemble, int64(8))

		actual, total, err = dataBase.GetTriggersSearchResults(searchResultsID, 0, 10)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, searchResults)
		So(total, ShouldResemble, int64(8))

		actual, total, err = dataBase.GetTriggersSearchResults(searchResultsID, 10, 20)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)
		So(total, ShouldResemble, int64(8))

		actual, total, err = dataBase.GetTriggersSearchResults(searchResultsID, 0, 3)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)
		So(actual, ShouldResemble, searchResults[:3])
		So(total, ShouldResemble, int64(8))

		actual, total, err = dataBase.GetTriggersSearchResults(searchResultsID, 1, 3)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 3)
		So(actual, ShouldResemble, searchResults[3:6])
		So(total, ShouldResemble, int64(8))

		actualExists, err := dataBase.IsTriggersSearchResultsExist(searchResultsID)
		So(err, ShouldBeNil)
		So(actualExists, ShouldBeTrue)

		actualExists, err = dataBase.IsTriggersSearchResultsExist("nonexistentPagerID")
		So(err, ShouldBeNil)
		So(actualExists, ShouldBeFalse)

		err = dataBase.DeleteTriggersSearchResults(searchResultsID)
		So(err, ShouldBeNil)

		actualExists, err = dataBase.IsTriggersSearchResultsExist(searchResultsID)
		So(err, ShouldBeNil)
		So(actualExists, ShouldBeFalse)
	})
}

func BenchmarkSaveTriggersSearchResults(b *testing.B) {
	logger = &logging.Logger{
		Logger: zerolog.New(&strings.Builder{}).With().Str(logging.ModuleFieldName, "dataBase").Logger(),
	}

	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	b.ReportAllocs()
	limits := []int{10, 100, 1000, 10000, 100000}
	for _, limit := range limits {
		data := make([]*moira.SearchResult, limit)
		for i := 0; i < limit; i++ {
			data[i] = &moira.SearchResult{
				ObjectID: "test",
				Highlights: []moira.SearchHighlight{
					{
						Field: "field1",
						Value: "Value1",
					},
					{
						Field: "field2",
						Value: "Value2",
					},
					{
						Field: "field3",
						Value: "Value3",
					},
				},
			}
		}
		dataBase.flush()
		b.Run(fmt.Sprintf("Benchmark%d", limit), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				dataBase.SaveTriggersSearchResults(fmt.Sprintf("test_%d_%d", limit, n), data) //nolint
			}
		})
	}
}
