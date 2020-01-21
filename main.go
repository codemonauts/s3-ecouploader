package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	BUCKET = "ecouploader-test"
	REGION = "eu-central-1"
)

func main() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-central-1"),
	}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	filename := "testdata/file.txt"
	shouldUpload := false

	f, err := os.Open(filename)
	if err != nil {
		fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	svc := s3.New(sess)
	input := &s3.HeadObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(filename),
	}

	result, err := svc.HeadObject(input)
	if err != nil {
		fmt.Println("file unavailable")
		shouldUpload = true
	} else {
		fmt.Println("file exists")
		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			fmt.Errorf("Cant hash file: %s", err)
		}
		s3hash := *result.ETag
		if s3hash[1:len(s3hash)-1] == hex.EncodeToString(h.Sum(nil)) {
			fmt.Println("file is the same")
			shouldUpload = false
		} else {
			fmt.Println("file changed. will upload")
			fmt.Println(result)
			fmt.Printf("%T\n", *result.ETag)
			fmt.Printf("%T\n", hex.EncodeToString(h.Sum(nil)))
			shouldUpload = true
		}

	}

	if shouldUpload {
		f.Seek(0, 0)
		// Upload the file to S3.
		uploadRes, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(BUCKET),
			Key:    aws.String(filename),
			Body:   f,
		})
		if err != nil {
			fmt.Errorf("failed to upload file, %v", err)
		}
		fmt.Printf("file uploaded to, %s\n", uploadRes.Location)
	} else {
		fmt.Println("Skipping this file")
	}

}
