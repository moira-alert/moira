package retries

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig_Validate(t *testing.T) {
	Convey("Test validating retries config", t, func() {
		type testcase struct {
			caseDesc    string
			conf        Config
			expectedErr error
		}

		var (
			testRetriesCount   uint64 = 10
			testMaxElapsedTIme        = time.Second * 10
		)

		cases := []testcase{
			{
				caseDesc:    "with empty config",
				conf:        Config{},
				expectedErr: errors.Join(errNoInitialInterval, errNoMaxInterval, errNoMaxElapsedTimeAndMaxRetriesCount),
			},
			{
				caseDesc: "with only InitialInterval set",
				conf: Config{
					InitialInterval: testInitialInterval,
				},
				expectedErr: errors.Join(errNoMaxInterval, errNoMaxElapsedTimeAndMaxRetriesCount),
			},
			{
				caseDesc: "with only MaxInterval set",
				conf: Config{
					MaxInterval: testMaxInterval,
				},
				expectedErr: errors.Join(errNoInitialInterval, errNoMaxElapsedTimeAndMaxRetriesCount),
			},
			{
				caseDesc: "with only MaxRetriesCount set",
				conf: Config{
					MaxRetriesCount: testRetriesCount,
				},
				expectedErr: errors.Join(errNoInitialInterval, errNoMaxInterval),
			},
			{
				caseDesc: "with only MaxElapsedTime set",
				conf: Config{
					MaxElapsedTime: testMaxElapsedTIme,
				},
				expectedErr: errors.Join(errNoInitialInterval, errNoMaxInterval),
			},
			{
				caseDesc: "with valid config",
				conf: Config{
					InitialInterval: testInitialInterval,
					MaxInterval:     testMaxInterval,
					MaxElapsedTime:  testMaxElapsedTIme,
				},
				expectedErr: nil,
			},
		}

		for i := range cases {
			Convey(fmt.Sprintf("Case %d: %s", i+1, cases[i].caseDesc), func() {
				err := cases[i].conf.Validate()

				So(err, ShouldResemble, cases[i].expectedErr)
			})
		}
	})
}
