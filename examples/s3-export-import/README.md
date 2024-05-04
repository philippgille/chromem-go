# S3 Export/Import

This example shows how to export the DB to and import it from any S3-compatible blob storage service.

- The example uses [MinIO](https://github.com/minio/minio), but any S3-compatible storage works.
- The example uses [gocloud.dev](https://github.com/google/go-cloud) Go "Cloud Development Kit" from Google for interfacing with any S3-compatible storage, and because it provides methods for creating writers and readers that make it easy to use with `chromem-go`.

## How to run

1. Prepare the S3-compatible storage
   1. `docker run -d --rm --name minio -p 127.0.0.1:9000:9000 -p 127.0.0.1:9001:9001 quay.io/minio/minio:RELEASE.2024-05-01T01-11-10Z server /data --console-address ":9001"`
   2. Open the MinIO Console in your browser: <http://localhost:9001>
   3. Log in with user `minioadmin` and password `minioadmin`
   4. Use the web UI to create a bucket named `mybucket`
2. Set the OpenAI API key in your env as `OPENAI_API_KEY`
3. `go run .`

You can also check <http://localhost:9001/browser/mybucket> and see the exported DB as `chromem.gob.gz`.

To stop the MinIO server run `docker stop minio`.

## Output

```text
2024/05/04 19:24:07 Successfully exported DB to S3 storage.
2024/05/04 19:24:07 Imported collection with 1 documents
2024/05/04 19:24:07 Successfully imported DB from S3 storage.
```
