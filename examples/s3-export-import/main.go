package main

import (
	"context"
	"log"
	"os"

	"github.com/philippgille/chromem-go"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	ctx := context.Background()

	// As S3-style storage we use a local MinIO instance in this example. It has
	// default credentials which we set in environment variables in order to be
	// read when calling `blob.OpenBucket`, because that call implies using the
	// AWS SDK default "shared config" loader.
	// That loader checks the environment variables, but can also fall back to a
	// file in `~/.aws/config` from `aws sso login` or similar.
	// See https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
	// for details about credential loading.
	err := os.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
	if err != nil {
		panic(err)
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")
	if err != nil {
		panic(err)
	}
	// A region configuration is also required. Alternatively it can be passed in
	// the connection string with "&region=us-west-1" for example.
	err = os.Setenv("AWS_DEFAULT_REGION", "us-west-1")
	if err != nil {
		panic(err)
	}

	// Export DB
	err = exportDB(ctx)
	if err != nil {
		panic(err)
	}
	log.Println("Successfully exported DB to S3 storage.")

	// Import DB
	err = importDB(ctx)
	if err != nil {
		panic(err)
	}
	log.Println("Successfully imported DB from S3 storage.")
}

func exportDB(ctx context.Context) error {
	// Create and fill DB
	db := chromem.NewDB()
	c, err := db.CreateCollection("knowledge-base", nil, nil)
	if err != nil {
		return err
	}
	err = c.AddDocument(ctx, chromem.Document{
		ID:      "1",
		Content: "The sky is blue because of Rayleigh scattering.",
	})
	if err != nil {
		return err
	}

	// Open S3 bucket. We're using a local MinIO instance here, but it can be any
	// S3-compatible storage. We're also using the gocloud.dev/blob package instead
	// of the AWS SDK for Go directly, because it provides a unified Writer/Reader
	// API for different cloud storage providers.
	bucket, err := blob.OpenBucket(ctx, "s3://mybucket?"+
		"endpoint=localhost:9000&"+
		"disableSSL=true&"+
		"s3ForcePathStyle=true")
	if err != nil {
		return err
	}

	// Create writer to an S3 object
	w, err := bucket.NewWriter(ctx, "chromem.gob.gz", nil)
	if err != nil {
		return err
	}
	// Instead of deferring w.Close() here, we close it at the end of the function
	// to handle its errors, as the close is important for the actual write to happen
	// and can lead to errors such as "The specified bucket does not exist" etc.
	// Another option is to use a named return value and defer a function that
	// overwrites the error with the close error or uses [errors.Join] or similar.

	// Persist the DB to the S3 object
	err = db.ExportToWriter(w, true, "")
	if err != nil {
		return err
	}

	return w.Close()
}

func importDB(ctx context.Context) error {
	// Open S3 bucket. We're using a local MinIO instance here, but it can be any
	// S3-compatible storage. We're also using the gocloud.dev/blob package instead
	// of the AWS SDK for Go directly, because it provides a unified Writer/Reader
	// API for different cloud storage providers.
	bucket, err := blob.OpenBucket(ctx, "s3://mybucket?"+
		"endpoint=localhost:9000&"+
		"disableSSL=true&"+
		"s3ForcePathStyle=true")
	if err != nil {
		return err
	}

	// Open reader to the S3 object
	r, err := bucket.NewReader(ctx, "chromem.gob.gz", nil)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create empty DB
	db := chromem.NewDB()

	// Import the DB from the S3 object
	err = db.ImportFromReader(r, "")
	if err != nil {
		return err
	}

	c := db.GetCollection("knowledge-base", nil)
	log.Printf("Imported collection with %d documents\n", c.Count())

	return nil
}
