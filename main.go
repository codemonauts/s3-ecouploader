package main

import (
	"bufio"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/peak/s3hash"
)

type Statistics struct {
	Start     time.Time
	End       time.Time
	Duration  time.Duration
	FileCount int
	New       int
	Changed   int
}

const (
	CHUNK     = 5
	BYTESINMB = 1024 * 1024
)

var (
	SESS       *session.Session
	UPLOADER   *s3manager.Uploader
	BUCKET     string
	REGION     string
	SRC        string
	DEST       string
	STATISTICS Statistics
)

func printStatistics() {
	skipped := (STATISTICS.FileCount) - ((STATISTICS.Changed) + (STATISTICS.New))

	fmt.Println("##############################")
	fmt.Printf("Start Time: %s\n", STATISTICS.Start.Format(time.RFC3339))
	fmt.Printf("  End Time: %s\n", STATISTICS.End.Format(time.RFC3339))
	fmt.Printf("  Duration: %s\n", STATISTICS.Duration)
	fmt.Println("")
	fmt.Printf("  Total File Count: %d\n", STATISTICS.FileCount)
	fmt.Printf("    Uploaded (New): %d\n", STATISTICS.New)
	fmt.Printf("Uploaded (Changed): %d\n", STATISTICS.Changed)
	fmt.Printf("           Skipped: %d\n", skipped)
	fmt.Println("##############################")
}
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

func buildRemotePath(path string, src string, dest string) string {
	relativePath := strings.Replace(path, src, "", 1)
	remotePath := fmt.Sprintf("%s%s", dest, relativePath)
	return remotePath
}

func checkFile(path string, force bool) error {
	STATISTICS.FileCount++
	remotePath := buildRemotePath(path, SRC, DEST)
	if force {
		uploadFile(path, remotePath)
		return nil
	}

	log.Info(path)
	s3Hash, err := getS3ETag(remotePath)
	if err != nil {
		log.Debug("File doesn't exists in S3 -> Uploading")
		STATISTICS.New++
		uploadFile(path, remotePath)
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
		STATISTICS.Changed++
		uploadFile(path, remotePath)
	}
	return nil
}

func handler(path string, f os.FileInfo, err error, force bool) error {
	if !f.IsDir() { // Only check files
		return checkFile(path, force)
	}
	return nil
}

func uploadFile(path string, remotePath string) {
	log.Info("Uploading ", path)
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		log.Errorf("failed to open file %q, %v", path, err)
		return
	}

	// Upload the file to S3.
	uploadRes, err := UPLOADER.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(remotePath),
		Body:   f,
	})
	if err != nil {
		log.Errorf("failed to upload file, %v", err)
		return
	}
	log.Debugf("File uploaded to %s\n", uploadRes.Location)
}

func main() {
	flag.StringVar(&BUCKET, "bucket", "", "Destination S3 Bucket")
	flag.StringVar(&REGION, "region", "", "Region of the S3 Bucket")
	flag.StringVar(&SRC, "src", "", "Local folder to backup")
	flag.StringVar(&DEST, "dest", "", "Remote prefix for S3")
	forcePtr := flag.Bool("force", false, "Skip hashing and upload all files")
	debugPtr := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if BUCKET == "" || REGION == "" || SRC == "" {
		log.Fatal("bucket, region and src are all required parameters")
	}

	if *debugPtr {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *forcePtr {
		log.Info("Will force upload every file due to -force flag")
	}

	if _, err := os.Stat(SRC); os.IsNotExist(err) {
		log.Fatalf("The folder %q doesn't exists", SRC)
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

	log.Infof("Starting to scan %q\n", SRC)
	STATISTICS.Start = time.Now()

	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	var fileList []string
	if info.Mode()&os.ModeCharDevice == 0 || info.Size() > 0 {
		reader := bufio.NewReader(os.Stdin)

		for {
			line, _, e := reader.ReadLine()
			if e != nil && e == io.EOF {
				break
			}
			fileList = append(fileList, string(line))
		}
	}

	if len(fileList) > 0 {
		log.Infof("Got %d files from stdin. Starting to check them\n", len(fileList))
		for _, path := range fileList {
			fi, err := os.Stat(path)
			if err != nil {
				log.Errorf("failed to open file %q, %v", path, err)
				continue
			}
			if !fi.IsDir() {
				checkFile(path, *forcePtr)
			}
		}
	} else {
		log.Infof("Got no file list from stdin. Starting to walk %q\n", SRC)
		err = filepath.Walk(SRC, func(path string, info os.FileInfo, err error) error {
			return handler(path, info, err, *forcePtr)
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	STATISTICS.End = time.Now()
	STATISTICS.Duration = time.Since(STATISTICS.Start)
	log.Info("Finished")

	printStatistics()
}
