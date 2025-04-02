package webhook

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_prepareDeliveryCheck(t *testing.T) {
	Convey("Test prepareDeliveryCheck", t, func() {
		Convey("with error after populating url template", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id "

			actualCheckData, err := prepareDeliveryCheck(testContact, nil, urlTemplate, testTrigger.ID)
			So(err, ShouldNotBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with empty rsp map", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(testContact, map[string]interface{}{}, urlTemplate, testTrigger.ID)
			So(err, ShouldBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{
				URL:           "https://example.com/",
				Contact:       testContact,
				TriggerID:     testTrigger.ID,
				AttemptsCount: 0,
			})
		})

		Convey("with non empty rsp map", func() {
			urlTemplate := "https://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{
				URL:           "https://example.com/123456",
				Contact:       testContact,
				TriggerID:     testTrigger.ID,
				AttemptsCount: 0,
			})
		})

		Convey("with not url format", func() {
			urlTemplate := "example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldNotBeNil)
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with bad scheme", func() {
			urlTemplate := "smtp://example.com/{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldResemble, fmt.Errorf("got bad url for check request: %w, url: smtp://example.com/123456", fmt.Errorf("bad url scheme: smtp")))
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})

		Convey("with empty host", func() {
			urlTemplate := "https:///{{ .SendAlertResponse.request_id }}"

			actualCheckData, err := prepareDeliveryCheck(
				testContact,
				map[string]interface{}{
					"request_id": 123456,
				},
				urlTemplate,
				testTrigger.ID)
			So(err, ShouldResemble, fmt.Errorf("got bad url for check request: %w, url: https:///123456", fmt.Errorf("host is empty")))
			So(actualCheckData, ShouldResemble, deliveryCheckData{})
		})
	})
}
