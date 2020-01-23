package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/peak/s3hash"
)

const (
	CHUNK     = 5
	BYTESINMB = 1024 * 1024
)

var (
	SESS     *session.Session
	UPLOADER *s3manager.Uploader
	BUCKET   string
	REGION   string
	FOLDER   string
)

func getS3ETag(key string) (string, error) {
	log.Debugf("Checking if %q exists in S3", key)
	svc := s3.New(SESS)
	input := &s3.HeadObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(key),
	}

	result, err := svc.HeadObject(input)
	if err != nil {
		return "", err
	} else {
		return *result.ETag, nil
	}
}

func handler(path string, f os.FileInfo, err error, force bool) error {
	if !f.IsDir() { // Only check files
		if force {
			uploadFile(path)
			return nil
		}

		log.Info(path)
		s3Hash, err := getS3ETag(path)
		if err != nil {
			log.Debug("File doesn't exists in S3 -> Uploading")
			uploadFile(path)
			return nil
		}

		localHash, err := s3hash.CalculateForFile(path, int64(CHUNK*BYTESINMB))
		if err != nil {
			return err
		}

		// Strip quotes from string
		s3Hash = s3Hash[1 : len(s3Hash)-1]

		if s3Hash == localHash {
			log.Debug("File didn't changed -> Skipping")
		} else {
			log.WithFields(log.Fields{
				"s3Hash":    s3Hash,
				"localHash": localHash,
			}).Debug("File changed -> Uploading.")
			uploadFile(path)
		}
	}
	return nil
}

func uploadFile(path string) {
	log.Info("Uploading ", path)
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		log.Error("failed to open file %q, %v", path, err)
		return
	}

	// Upload the file to S3.
	uploadRes, err := UPLOADER.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(path),
		Body:   f,
	})
	if err != nil {
		log.Error("failed to upload file, %v", err)
		return
	}
	log.Debugf("File uploaded to %s\n", uploadRes.Location)
}

func main() {
	flag.StringVar(&BUCKET, "bucket", "", "Destination S3 Bucket")
	flag.StringVar(&REGION, "region", "", "Region of the S3 Bucket")
	flag.StringVar(&FOLDER, "folder", "", "Local folder to backup")
	forcePtr := flag.Bool("force", false, "Skip hashing and upload all files")
	debugPtr := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if BUCKET == "" || REGION == "" || FOLDER == "" {
		log.Fatal("bucket, region and folder are all required parameters")
	}

	if *debugPtr {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *forcePtr {
		log.Info("Will force upload every file due to -force flag")
	}

	if _, err := os.Stat(FOLDER); os.IsNotExist(err) {
		log.Fatalf("The folder %q doesn't exists", FOLDER)
	}

	log.Info("Creating S3 session")
	SESS = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	}))

	values, _ := SESS.Config.Credentials.Get()
	if !values.HasKeys() {
		log.Fatal("Can't find valid AWS credentials")
	}

	log.Info("Creating S3 upload manager")
	UPLOADER = s3manager.NewUploader(SESS, func(u *s3manager.Uploader) {
		u.PartSize = CHUNK * BYTESINMB
	})

	log.Infof("Starting to scan %q\n", FOLDER)
	err := filepath.Walk(FOLDER, func(path string, info os.FileInfo, err error) error {
		return handler(path, info, err, *forcePtr)
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Finished")

}
