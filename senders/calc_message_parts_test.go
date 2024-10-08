package senders

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCalculateMessagePartsLength(t *testing.T) {
	Convey("Message parts length calculation tests", t, func() {
		Convey("descLen+eventsLen <= maxChars", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 20, 78)
			So(descNewLen, ShouldResemble, 20)
			So(eventsNewLen, ShouldResemble, 78)
		})

		Convey("descLen > maxChars/2 && eventsLen <= maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 40)
			So(descNewLen, ShouldResemble, 50)
			So(eventsNewLen, ShouldResemble, 40)
		})

		Convey("eventsLen > maxChars/2 && descLen <= maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 40, 70)
			So(descNewLen, ShouldResemble, 40)
			So(eventsNewLen, ShouldResemble, 60)
		})

		Convey("Both greater than maxChars/2", func() {
			descNewLen, eventsNewLen := CalculateMessagePartsLength(100, 70, 70)
			So(descNewLen, ShouldResemble, 40)
			So(eventsNewLen, ShouldResemble, 50)
		})
	})
}

func TestCalculateMessagePartsBetweenTagsDescEvents(t *testing.T) {
	Convey("Message parts calculating test (for tags, desc, events)", t, func() {
		type given struct {
			maxChars  int
			tagsLen   int
			descLen   int
			eventsLen int
		}

		type expected struct {
			tagsLen   int
			descLen   int
			eventsLen int
		}

		type testcase struct {
			given       given
			expected    expected
			description string
		}

		cases := []testcase{
			{
				description: "with maxChars < 0",
				given: given{
					maxChars:  -1,
					tagsLen:   10,
					descLen:   10,
					eventsLen: 10,
				},
				expected: expected{
					tagsLen:   0,
					descLen:   0,
					eventsLen: 0,
				},
			},
			{
				description: "with tagsLen + descLen + eventsLen <= maxChars",
				given: given{
					maxChars:  100,
					tagsLen:   20,
					descLen:   50,
					eventsLen: 30,
				},
				expected: expected{
					tagsLen:   20,
					descLen:   50,
					eventsLen: 30,
				},
			},
			{
				description: "with tagsLen > maxChars/3, descLen and eventsLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   50,
					descLen:   30,
					eventsLen: 30,
				},
				expected: expected{
					tagsLen:   40,
					descLen:   30,
					eventsLen: 30,
				},
			},
			{
				description: "with descLen > maxChars/3, tagsLen and eventsLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   30,
					descLen:   50,
					eventsLen: 31,
				},
				expected: expected{
					tagsLen:   30,
					descLen:   39,
					eventsLen: 31,
				},
			},
			{
				description: "with eventsLen > maxChars/3, tagsLen and descLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   33,
					descLen:   33,
					eventsLen: 61,
				},
				expected: expected{
					tagsLen:   33,
					descLen:   33,
					eventsLen: 34,
				},
			},
			{
				description: "with tagsLen and descLen > maxChars/3, eventsLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   55,
					descLen:   46,
					eventsLen: 31,
				},
				expected: expected{
					tagsLen:   33,
					descLen:   36,
					eventsLen: 31,
				},
			},
			{
				description: "with tagsLen and eventsLen > maxChars/3, descLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   55,
					descLen:   33,
					eventsLen: 100,
				},
				expected: expected{
					tagsLen:   33,
					descLen:   33,
					eventsLen: 34,
				},
			},
			{
				description: "with descLen and eventsLen > maxChars/3, tagsLen <= maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   29,
					descLen:   56,
					eventsLen: 100,
				},
				expected: expected{
					tagsLen:   29,
					descLen:   35,
					eventsLen: 35,
				},
			},
			{
				description: "with tagsLen, descLen and eventsLen > maxChars/3",
				given: given{
					maxChars:  100,
					tagsLen:   55,
					descLen:   40,
					eventsLen: 100,
				},
				expected: expected{
					tagsLen:   33,
					descLen:   33,
					eventsLen: 33,
				},
			},
			{
				description: "with tagsLen, descLen > maxChars/3, eventsLen <= maxChars/3 and maxChars - maxChars/3 - eventsLen > descLen",
				given: given{
					maxChars:  100,
					tagsLen:   100,
					descLen:   34,
					eventsLen: 20,
				},
				expected: expected{
					tagsLen:   33,
					descLen:   34,
					eventsLen: 20,
				},
			},
		}

		for i, c := range cases {
			Convey(fmt.Sprintf("case %d: %s", i+1, c.description), func() {
				tagsNewLen, descNewLen, eventsNewLen := CalculateMessagePartsBetweenTagsDescEvents(c.given.maxChars, c.given.tagsLen, c.given.descLen, c.given.eventsLen)

				So(tagsNewLen, ShouldResemble, c.expected.tagsLen)
				So(descNewLen, ShouldResemble, c.expected.descLen)
				So(eventsNewLen, ShouldResemble, c.expected.eventsLen)
			})
		}
	})
}
