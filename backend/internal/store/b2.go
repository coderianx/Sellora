package store

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

var B2Client *s3.S3

func ConnectB2() {
	required := []string{
		"B2_REGION",
		"B2_ENDPOINT",
		"B2_KEY_ID",
		"B2_APPLICATION_KEY",
		"B2_BUCKET",
	}

	for _, key := range required {
		if len(os.Getenv(key)) == 0 {
			fmt.Println("[ERROR]", key, "is not set in .env file")
			os.Exit(1)
		}
	}

	B2Client = NewB2()

	fmt.Println("[INFO] B2 connected")
}

func NewB2() *s3.S3 {
	cfg := &aws.Config{
		Region: aws.String(os.Getenv("B2_REGION")),

		Endpoint: aws.String(
			os.Getenv("B2_ENDPOINT"),
		),

		Credentials: credentials.NewStaticCredentials(
			os.Getenv("B2_KEY_ID"),
			os.Getenv("B2_APPLICATION_KEY"),
			"",
		),

		S3ForcePathStyle: aws.Bool(true),
	}

	sess := session.Must(
		session.NewSession(cfg),
	)

	return s3.New(sess)
}

func UploadFile(
	client *s3.S3,
	file multipart.File,
	filename string,
	contentType string,
) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectKey := fmt.Sprintf(
		"uploads/%s/%s",
		uuid.New().String(),
		cleanFilename(filename),
	)

	_, err := client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("B2_BUCKET")),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	return objectKey, nil
}

func DeleteFile(
	client *s3.S3,
	objectKey string,
) error {
	_, err := client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(os.Getenv("B2_BUCKET")),
		Key:    aws.String(objectKey),
	})

	return err
}

func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))

	if name == "." || name == "" || name == string(filepath.Separator) {
		return "file"
	}

	return name
}
