package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// ImageStore implements the ImageStore interface for aws s3
type ImageStore struct {
	sess     *session.Session
	uploader *s3manager.Uploader
	bucket   string
}

// Init initializes the s3 image store with config from the yaml file
func (imageStore *ImageStore) Init(imageStoreSettings map[string]string) error {
	config := &aws.Config{}

	accessKeyID := imageStoreSettings["access_key_id"]
	if accessKeyID == "" {
		return fmt.Errorf("access key id not found while configuring s3 image store")
	}
	accessKey := imageStoreSettings["access_key"]
	if accessKey == "" {
		return fmt.Errorf("access key not found while configuring s3 image store")
	}
	config.Credentials = credentials.NewStaticCredentials(accessKeyID, accessKey, "")

	region := imageStoreSettings["region"]
	if region == "" {
		return fmt.Errorf("region not found while configuring s3 image store")
	}
	config.Region = aws.String(region)

	imageStore.bucket = imageStoreSettings["bucket"]
	if imageStore.bucket == "" {
		return fmt.Errorf("bucket not found while configuring s3 image store")
	}

	var err error
	imageStore.sess, err = session.NewSession(config)
	if err != nil {
		return fmt.Errorf("could not configure s3 session: %s", err)
	}
	imageStore.uploader = s3manager.NewUploader(imageStore.sess)

	return nil
}
