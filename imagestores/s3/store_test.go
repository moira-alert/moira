package s3

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildUploadInput(t *testing.T) {
	imageStore := &ImageStore{}
	imageStore.Init(map[string]string{
		"access_key":    "123",
		"access_key_id": "123",
		"bucket":        "testbucket",
		"region":        "ap-south-1",
	})
	Convey("Build S3 upload input tests", t, func() {
		Convey("Build upload input with empty byte slice", func() {
			uploadInput := imageStore.buildUploadInput([]byte{})
			So(uploadInput.Body, ShouldResemble, bytes.NewReader([]byte{}))
			So(uploadInput.Bucket, ShouldResemble, aws.String(imageStore.bucket))
			So(uploadInput.ACL, ShouldResemble, aws.String("public-read"))
		})
	})
}
