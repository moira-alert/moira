package senders

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestReadImageStoreConfig(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockimageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
	logger := mock_moira_alert.NewMockLogger(mockCtrl)
	imageStores := map[string]moira.ImageStore{
		"s3": mockimageStore,
	}

	Convey("Read image store config tests", t, func() {
		Convey("no image_store in settings", func() {
			logger.EXPECT().Warningf("Cannot read image_store from the config, will not be able to attach plot images to alerts")
			imageStoreID, imageStore, imageStoreConfigured :=
				ReadImageStoreConfig(map[string]string{}, imageStores, logger)
			So(imageStoreConfigured, ShouldResemble, false)
			So(imageStoreID, ShouldResemble, "")
			So(imageStore, ShouldResemble, nil)
		})

		Convey("wrong image store name", func() {
			logger.EXPECT().Warningf("Image store specified (%s) has not been configured", "s4")
			imageStoreID, imageStore, imageStoreConfigured :=
				ReadImageStoreConfig(map[string]string{"image_store": "s4"}, imageStores, logger)
			So(imageStoreConfigured, ShouldResemble, false)
			So(imageStoreID, ShouldResemble, "")
			So(imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			logger.EXPECT().Warningf("Image store specified (%s) has not been configured", "s3")
			mockimageStore.EXPECT().IsEnabled().Return(false)
			imageStoreID, imageStore, imageStoreConfigured :=
				ReadImageStoreConfig(map[string]string{"image_store": "s3"}, imageStores, logger)
			So(imageStoreConfigured, ShouldResemble, false)
			So(imageStoreID, ShouldResemble, "")
			So(imageStore, ShouldResemble, nil)
		})

		Convey("image store is configured", func() {
			mockimageStore.EXPECT().IsEnabled().Return(true)
			imageStoreID, imageStore, imageStoreConfigured :=
				ReadImageStoreConfig(map[string]string{"image_store": "s3"}, imageStores, logger)
			So(imageStoreConfigured, ShouldResemble, true)
			So(imageStoreID, ShouldResemble, "s3")
			So(imageStore, ShouldResemble, mockimageStore)
		})
	})
}
