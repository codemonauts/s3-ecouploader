# s3-ecouploader

This tool was created to run on an embedded low-end NAS of one of our customers which was running some kind of very limited
Linux so we decided to use Go and compile a static binary. To also save traffic and stoarge costs, this tool first
checks if a file exists in the destination bucket. If so, it also checks the `ETag` provided by S3 (which is just the
MD5 hash of this file). This way we only upload a file if it is not present in the bucket or changed since the last
upload.

## Credentials
The tool will read it's credentials from the default awscli config file located at `~/.aws/credentials`.

## Usage
```
  -bucket string
        Destination S3 Bucket
  -region string
        Region of the S3 Bucket
  -src string
        Local folder to backup
  -dest string
        Remote prefix for S3
  -debug
        Enable debug logging
  -force
        Skip hashing and upload all files
  -stdin
        Read a list of files from stdin
```

## Optimize runtime for very big folders
When running this tool every day on a big folder, most of the files didn't got touched since last run and therefore
don't need to be processed by this tool. One can now use `find` to get a list of files which got modified a short while
ago. This way the amount of files which have to be processed by this tool drops significantly. We run our backup
scripts every night and start the script like this:
```
find /mnt/data -mtime -2 -type f | ./s3-ecouploader -stdin ....
```
This command will process all files (`-type f`) which have a modification timestamp (`mtime`) between now and two days ago
(`-2`).

