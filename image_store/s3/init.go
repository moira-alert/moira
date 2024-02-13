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
	enabled  bool
}

// Init initializes the s3 image store with config from the yaml file
func (imageStore *ImageStore) Init(config Config) error {
	awsconfig := &aws.Config{}

	if config.AccessKeyID == "" {
		return fmt.Errorf("access key id not found while configuring s3 image store")
	}
	if config.AccessKey == "" {
		return fmt.Errorf("access key not found while configuring s3 image store")
	}
	awsconfig.Credentials = credentials.NewStaticCredentials(config.AccessKeyID, config.AccessKey, "")

	if config.Region == "" {
		return fmt.Errorf("region not found while configuring s3 image store")
	}
	awsconfig.Region = aws.String(config.Region)

	imageStore.bucket = config.Bucket
	if imageStore.bucket == "" {
		return fmt.Errorf("bucket not found while configuring s3 image store")
	}

	var err error
	imageStore.sess, err = session.NewSession(awsconfig)
	if err != nil {
		return fmt.Errorf("could not configure s3 session: %w", err)
	}
	imageStore.uploader = s3manager.NewUploader(imageStore.sess)

	imageStore.enabled = true
	return nil
}

// IsEnabled indicates whether the image store has been configured or not
func (imageStore *ImageStore) IsEnabled() bool {
	return imageStore.enabled
}
