package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_applyPaginate(t *testing.T) {
	Convey("Test paginating", t, func() {
		entities := make([]int, 0, 'z'-'a'+1)
		for i := 0; i < 40; i++ {
			entities = append(entities, i)
		}

		type testcase struct {
			page             int64
			size             int64
			total            int64
			givenEntities    []int
			expectedEntities []int
			desc             string
		}

		cases := []testcase{
			{
				page:             -1,
				givenEntities:    entities,
				expectedEntities: []int{},
				desc:             "with negative page",
			},
			{
				page:             1,
				size:             -1,
				givenEntities:    entities,
				expectedEntities: []int{},
				desc:             "with positive page and negative size",
			},
			{
				page:             1,
				size:             10,
				total:            7,
				givenEntities:    entities,
				expectedEntities: []int{},
				desc:             "out of range",
			},
			{
				page:             0,
				size:             -1,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: entities,
				desc:             "fetch all entities",
			},
			{
				page:             0,
				size:             -2,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: entities,
				desc:             "again fetch all entities",
			},
			{
				page:             0,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{0, 1, 2, 3, 4, 5, 6},
				desc:             "first page with size 7 (page = 0)",
			},
			{
				page:             1,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{7, 8, 9, 10, 11, 12, 13},
				desc:             "second page with size 7 (page = 1)",
			},
			{
				page:             2,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{14, 15, 16, 17, 18, 19, 20},
				desc:             "third page with size 7 (page = 2)",
			},
			{
				page:             3,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{21, 22, 23, 24, 25, 26, 27},
				desc:             "forth page with size 7 (page = 3)",
			},
			{
				page:             4,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{28, 29, 30, 31, 32, 33, 34},
				desc:             "fifth page with size 7 (page = 4)",
			},
			{
				page:             5,
				size:             7,
				total:            int64(len(entities)),
				givenEntities:    entities,
				expectedEntities: []int{35, 36, 37, 38, 39},
				desc:             "last page with size 7 (page = 5)",
			},
		}

		for i, tcase := range cases {
			Convey(fmt.Sprintf("Case %v: %s", i+1, tcase.desc), func() {
				gotEntities := applyPagination[int](tcase.page, tcase.size, tcase.total, tcase.givenEntities)

				So(gotEntities, ShouldResemble, tcase.expectedEntities)
			})
		}
	})
}
