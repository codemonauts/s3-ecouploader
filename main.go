package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/peak/s3hash"
)

const (
	BUCKET    = "ecouploader-test"
	REGION    = "eu-central-1"
	FOLDER    = "testdata/"
	CHUNK     = 5
	bytesInMb = 1024 * 1024
)

var (
	sess     *session.Session
	uploader *s3manager.Uploader
)

func getS3ETag(key string) (string, error) {
	log.Printf("Looking for %q in S3", key)
	svc := s3.New(sess)
	input := &s3.HeadObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(key),
	}

	result, err := svc.HeadObject(input)
	if err != nil {
		return "", err
	} else {
		log.Print("file exists")
		return *result.ETag, nil
	}
}

func handler(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		log.Printf("Checking %q\n", path)
		s3Hash, err := getS3ETag(path)
		if err != nil {
			uploadFile(path)
			return nil
		}

		localHash, err := s3hash.CalculateForFile(path, int64(CHUNK*bytesInMb))
		if err != nil {
			log.Printf("Cant hash file: %s", err)
		}

		// Strip quotes from string
		s3Hash = s3Hash[1 : len(s3Hash)-1]

		if s3Hash == localHash {
			log.Println("File didn't changed. Skipping")
		} else {
			log.Println("File changed. Uploading")
			log.Printf("s3hash: %s", s3Hash)
			log.Printf("localhash: %s", localHash)
			uploadFile(path)
		}
	}
	return nil
}

func uploadFile(path string) {
	log.Println("Uploading ", path)
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		log.Printf("failed to open file %q, %v", path, err)
	}

	// Upload the file to S3.
	uploadRes, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(path),
		Body:   f,
	})
	if err != nil {
		log.Printf("failed to upload file, %v", err)
	}
	log.Printf("file uploaded to, %s\n", uploadRes.Location)
}

func main() {
	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	}))

	// Create an uploader with the session and default options
	uploader = s3manager.NewUploader(sess)

	err := filepath.Walk(FOLDER, handler)
	if err != nil {
		log.Println(err)
	}

}
