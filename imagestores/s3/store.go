package s3

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

// StoreImage stores an image in aws s3 and returns the link to it
func (imageStore *ImageStore) StoreImage(image []byte) (string, error) {
	uploadInput := imageStore.buildUploadInput(image)
	result, err := imageStore.uploader.Upload(uploadInput)
	if err != nil {
		return "", fmt.Errorf("error while uploading to s3: %s", err)
	}

	return result.Location, nil
}

func (imageStore *ImageStore) buildUploadInput(image []byte) *s3manager.UploadInput {
	key := "moira-plots/" + uuid.New().String()
	return &s3manager.UploadInput{
		Bucket:             aws.String(imageStore.bucket),
		Key:                aws.String(key),
		ACL:                aws.String("public-read"),
		Body:               bytes.NewReader(image),
		ContentType:        aws.String(http.DetectContentType(image)),
		ContentDisposition: aws.String("attachment"),
	}
}
