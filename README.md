# s3-ecouploader
Itterate a local folder and check every file if it is already in the bucket. If so, check MD5 beforehand
before streaming it to S3 (to save RAM).

## Workflow
```
	// init s3 client
	// for file in local folder
        // if file in s3:
            // compare md5 hash
                // if no match -> upload (new version)
        // else -> upload
```
