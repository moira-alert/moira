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
			config := Config{}
			err := imageStore.Init(config)
			So(err, ShouldResemble, fmt.Errorf("access key id not found while configuring s3 image store"))
			So(imageStore, ShouldResemble, &ImageStore{})
		})

		Convey("Missing access_key", func() {
			config := Config{
				AccessKeyID: "123",
				Region:      "ap-south-1",
				Bucket:      "testbucket",
			}
			err := imageStore.Init(config)
			So(err, ShouldResemble, fmt.Errorf("access key not found while configuring s3 image store"))
			So(imageStore, ShouldResemble, &ImageStore{})
		})

		Convey("Missing region", func() {
			config := Config{
				AccessKeyID: "123",
				AccessKey:   "123",
				Bucket:      "testbucket",
			}
			err := imageStore.Init(config)
			So(err, ShouldResemble, fmt.Errorf("region not found while configuring s3 image store"))
			So(imageStore, ShouldResemble, &ImageStore{})
		})

		Convey("Missing bucket", func() {
			config := Config{
				AccessKeyID: "123",
				AccessKey:   "123",
				Region:      "ap-south-1",
			}
			err := imageStore.Init(config)
			So(err, ShouldResemble, fmt.Errorf("bucket not found while configuring s3 image store"))
			So(imageStore, ShouldResemble, &ImageStore{})
		})

		Convey("Has settings", func() {
			config := Config{
				AccessKeyID: "123",
				AccessKey:   "123",
				Region:      "ap-south-1",
				Bucket:      "testbucket",
			}
			imageStore.Init(config) //nolint
			val, _ := imageStore.sess.Config.Credentials.Get()
			So(val.AccessKeyID, ShouldResemble, config.AccessKeyID)
			So(val.SecretAccessKey, ShouldResemble, config.AccessKey)
			So(imageStore.sess.Config.Region, ShouldResemble, aws.String(config.Region))
			So(imageStore.bucket, ShouldResemble, config.Bucket)
			So(imageStore.enabled, ShouldResemble, true)
		})
	})
}
