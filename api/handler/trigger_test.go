package handler

import (
	"bytes"
	"context"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/api/middleware"
	. "github.com/smartystreets/goconvey/convey"

	"net/http"
	"net/http/httptest"
	"testing"
)


func TestSimpleModeMultipleTargetsWarnValue (t *testing.T){
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
    "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "warn_value": 10,
  "trigger_type": "falling",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	requestRecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(updateTrigger)

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	ctx := request.Context()
	ctx = context.WithValue(ctx, middleware.ContextKey("triggerID"), "GraphiteStoragesFreeSpace")
	request = request.WithContext(ctx)

	handler.ServeHTTP(requestRecorder, request)

	Convey("Success update", t, func() {
		So(requestRecorder.Code, ShouldResemble, 500)
		So(requestRecorder.Body.String(), ShouldContainSubstring, "Can't use warn_value with multiple targets")
	})
}

func TestSimpleModeMultipleTargetsErrorValue (t *testing.T){
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var body bytes.Buffer
	body.WriteString(`{
  "id": "GraphiteStoragesFreeSpace",
  "name": "Graphite storage free space low",
  "desc": "Graphite ClickHouse",
  "targets": [
    "aliasByNode(DevOps.system.graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.sd2-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.bst-graphite01.disk.root.gigabyte_percentfree, 2, 4)",
    "aliasByNode(DevOps.system.dtl-graphite01.disk._mnt_data.gigabyte_percentfree, 2, 4)"
  ],
  "error_value": 5,
  "trigger_type": "falling",
  "tags": [
    "Normal",
    "DevOps",
    "DevOpsGraphite-duty"
  ],
  "ttl_state": "NODATA",
  "ttl": 600,
  "expression": "",
  "is_remote": false,
  "mute_new_metrics": false,
  "throttling": 0
	}`)

	requestRecorder := httptest.NewRecorder()
	handler := http.HandlerFunc(updateTrigger)

	request, _ := http.NewRequest("PUT", "/api/trigger", &body)
	request.Header.Set("Content-Type", "application/json")

	ctx := request.Context()
	ctx = context.WithValue(ctx, middleware.ContextKey("triggerID"), "GraphiteStoragesFreeSpace")
	request = request.WithContext(ctx)

	handler.ServeHTTP(requestRecorder, request)

	Convey("Success update", t, func() {
		So(requestRecorder.Code, ShouldResemble, 500)
		So(requestRecorder.Body.String(), ShouldContainSubstring, "Can't use error_value with multiple targets")
	})
}
