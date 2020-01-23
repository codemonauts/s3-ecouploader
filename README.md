# s3-ecouploader

This tool was created to run on an embedded NAS of one of our customers which was running some kind of very limited
Linux so we decided to use Go and compile a static binary. To also save traffic and stoarge costs, this tool first
checks if a file exists in the destination bucket. If so, it also checks the `ETag` provided by S3 (which is just the
MD5 hash of this file). This way we only upload a file if it is not present in the bucket or changed since the last
upload.

## Credentials
The tool will read it's credentials from the default awscli config file located at `~/.aws/credentials`.

## Usage
```
  -bucket string 
        Destination S3 Bucket (required)
  -region string
        Region of the S3 Bucket (required)
  -folder string
        Local folder to backup (required)
  -force
        Skip hashing and upload all files
  -debug
        Enable debug logging
```
