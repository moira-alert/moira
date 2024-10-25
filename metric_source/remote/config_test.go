package remote

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
	"testing"
	"time"

	"github.com/moira-alert/moira/metric_source/retries"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigWithValidateStruct(t *testing.T) {
	Convey("Test validating retries config", t, func() {
		type testcase struct {
			caseDesc string
			conf     Config
			errIsNil bool
		}

		var (
			testInitialInterval        = time.Second * 5
			testMaxInterval            = time.Second * 10
			testRetriesCount    uint64 = 10
			validatorErr               = validator.ValidationErrors{}
		)

		testRetriesConf := retries.Config{
			InitialInterval: testInitialInterval,
			MaxInterval:     testMaxInterval,
			MaxRetriesCount: testRetriesCount,
		}

		cases := []testcase{
			{
				caseDesc: "with empty config",
				conf:     Config{},
				errIsNil: false,
			},
			{
				caseDesc: "with retries config set",
				conf: Config{
					Retries:            testRetriesConf,
					HealthcheckRetries: testRetriesConf,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with retries config set and some url",
				conf: Config{
					URL:                "http://test-graphite",
					Retries:            testRetriesConf,
					HealthcheckRetries: testRetriesConf,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with retries config set, some url, timeout",
				conf: Config{
					Timeout:            time.Second,
					URL:                "http://test-graphite",
					Retries:            testRetriesConf,
					HealthcheckRetries: testRetriesConf,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with valid config",
				conf: Config{
					Timeout:            time.Second,
					HealthcheckTimeout: time.Millisecond,
					URL:                "http://test-graphite",
					Retries:            testRetriesConf,
					HealthcheckRetries: testRetriesConf,
				},
				errIsNil: true, //nil,
			},
		}

		for i := range cases {
			Convey(fmt.Sprintf("Case %d: %s", i+1, cases[i].caseDesc), func() {
				err := moira.ValidateStruct(cases[i].conf)

				if cases[i].errIsNil {
					So(err, ShouldBeNil)
				} else {
					So(errors.As(err, &validatorErr), ShouldBeTrue)
				}
			})
		}
	})
}
