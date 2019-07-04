package s3

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	Convey("Init tests", t, func() {
		imageStore := &ImageStore{}

		Convey("Empty settings", func() {
			imageStoreSettings := map[string]string{}
			err := imageStore.Init(imageStoreSettings)
			So(err, ShouldResemble, fmt.Errorf("access key id not found while configuring s3 image store"))
			So(imageStore, ShouldResemble, &ImageStore{})
		})

		Convey("Has settings", func() {
			imageStoreSettings := map[string]string{
				"access_key":    "123",
				"access_key_id": "123",
				"region":        "ap-south-1",
				"bucket":        "testbucket",
			}
			imageStore.Init(imageStoreSettings)
			val, _ := imageStore.sess.Config.Credentials.Get()
			So(val.AccessKeyID, ShouldResemble, imageStoreSettings["access_key_id"])
			So(val.SecretAccessKey, ShouldResemble, imageStoreSettings["access_key"])
			So(imageStore.sess.Config.Region, ShouldResemble, aws.String(imageStoreSettings["region"]))
			So(imageStore.bucket, ShouldResemble, imageStoreSettings["bucket"])
		})
	})
}
