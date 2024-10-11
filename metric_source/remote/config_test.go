package remote

import (
	"errors"
	"fmt"
	"github.com/moira-alert/moira/metric_source/retries"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig_validate(t *testing.T) {
	Convey("Test validating retries config", t, func() {
		type testcase struct {
			caseDesc    string
			conf        Config
			expectedErr error
		}

		var (
			testInitialInterval        = time.Second * 5
			testMaxInterval            = time.Second * 10
			testRetriesCount    uint64 = 10
		)

		testRetriesConf := retries.Config{
			InitialInterval: testInitialInterval,
			MaxInterval:     testMaxInterval,
			MaxRetriesCount: testRetriesCount,
		}

		cases := []testcase{
			{
				caseDesc:    "with empty config",
				conf:        Config{},
				expectedErr: errors.Join(errBadRemoteUrl, errNoTimeout, errNoHealthcheckTimeout, retries.Config{}.Validate(), retries.Config{}.Validate()),
			},
			{
				caseDesc: "with retries config set",
				conf: Config{
					Retries:            testRetriesConf,
					HealthcheckRetries: testRetriesConf,
				},
				expectedErr: errors.Join(errBadRemoteUrl, errNoTimeout, errNoHealthcheckTimeout),
			},
		}

		for i := range cases {
			Convey(fmt.Sprintf("Case %d: %s", i+1, cases[i].caseDesc), func() {
				err := cases[i].conf.validate()

				So(err, ShouldResemble, cases[i].expectedErr)
			})
		}
	})
}
