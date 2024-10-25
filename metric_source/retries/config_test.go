package retries

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/moira-alert/moira"
	"testing"
	"time"

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
			testRetriesCount   uint64 = 10
			testMaxElapsedTIme        = time.Second * 10
			validatorErr              = validator.ValidationErrors{}
		)

		cases := []testcase{
			{
				caseDesc: "with empty config",
				conf:     Config{},
				errIsNil: false,
			},
			{
				caseDesc: "with only InitialInterval set",
				conf: Config{
					InitialInterval: testInitialInterval,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with only MaxInterval set",
				conf: Config{
					MaxInterval: testMaxInterval,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with only MaxRetriesCount set",
				conf: Config{
					MaxRetriesCount: testRetriesCount,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with only MaxElapsedTime set",
				conf: Config{
					MaxElapsedTime: testMaxElapsedTIme,
				},
				errIsNil: false,
			},
			{
				caseDesc: "with valid config but only MaxElapsedTime set",
				conf: Config{
					InitialInterval: testInitialInterval,
					MaxInterval:     testMaxInterval,
					MaxElapsedTime:  testMaxElapsedTIme,
				},
				errIsNil: true,
			},
			{
				caseDesc: "with valid config but only MaxRetriesCount set",
				conf: Config{
					InitialInterval: testInitialInterval,
					MaxInterval:     testMaxInterval,
					MaxRetriesCount: testRetriesCount,
				},
				errIsNil: true,
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
