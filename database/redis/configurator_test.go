package redis

import (
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInitialization(t *testing.T) {
	Convey("Initialization methods", t, func() {
		mockCtrl := gomock.NewController(t)
		logger := mock_moira_alert.NewMockLogger(mockCtrl)
		config := Config{}
		database := Init(logger, config)
		So(database, ShouldNotBeEmpty)
		_, err := database.pool.Dial()
		So(err, ShouldNotBeNil)
	})
}
