package s3

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"
)

// StoreImage stores an image in aws s3 and returns the link to it
func (imageStore *ImageStore) StoreImage(image []byte) (string, error) {
	key := "moira-plots/" + uuid.New().String()
	uploadInput := &s3manager.UploadInput{
		Bucket:               aws.String(imageStore.bucket),
		Key:                  aws.String(key),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(image),
		ContentType:          aws.String(http.DetectContentType(image)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
		// StorageClass:         aws.String("INTELLIGENT_TIERING"),
	}
	result, err := imageStore.uploader.Upload(uploadInput)
	if err != nil {
		return "", fmt.Errorf("error while uploading to s3: %s", err)
	}

	return result.Location, nil
}
