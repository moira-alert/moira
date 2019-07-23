package victorops

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{}
		Convey("Empty map", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("cannot read the routing url from the yaml config"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"routing_url": "https://testurl.com",
				"front_uri":   "http://moira.uri",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.routingURL, ShouldResemble, "https://testurl.com")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sender.client, ShouldResemble, api.NewClient("https://testurl.com", nil))
		})
	})
}
