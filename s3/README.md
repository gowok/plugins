# `s3`

Manage multiple AWS S3 clients from Gowok configuration.

Each enabled entry under `s3` creates an S3 client.
Clients are stored by configuration key and can be retrieved with `s3.Client(name)`.

## Installation

```bash
go get github.com/gowok/plugins/s3
```

## Configuration

Register the plugin in your Gowok startup:

```go
package main

import (
	"context"
	"fmt"

	"github.com/gowok/gowok"
	"github.com/gowok/plugins/s3"
)

func main() {
	gowok.Get().Configures(
		s3.Configure(&s3.ConfigureInput{
			OnError: func(err error) {
				fmt.Println("S3 configuration error:", err)
			},
		}),
		run,
	).Run()
}

func run(ctx context.Context) {
	// Use the configured client here.
}
```

Define one or more named clients:

```yaml
s3:
  default:
    enabled: true
    region: us-east-1
    bucket: payments

  prod:
    enabled: true
    region: ap-southeast-1
    bucket: production-assets

  staging:
    enabled: false
    region: us-east-1
```

The map key is the client name.
`s3.Client()` looks up `default`; `s3.Client("prod")` looks up `prod`.

Only `enabled`, `region`, `endpoint`, and `bucket` are applied by the plugin.
Credentials come from the standard AWS SDK v2 credential chain, such as environment variables, shared AWS configuration, or an IAM role. Do not put credentials in committed YAML files.

## Usage

Retrieve a client and use the AWS SDK v2 S3 API:

```go
package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gowok/plugins/s3"
)

func run(ctx context.Context) {
	client := s3.Client().OrPanic()

	bucket := "payments"
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
	})
	if err != nil {
		fmt.Println("Failed to list objects:", err)
		return
	}

	for _, object := range output.Contents {
		fmt.Println(*object.Key)
	}
}
```

`Client` accepts multiple names and returns the first available client in the supplied order.
It returns an empty `some.Some` when none of the requested clients exists.

## Startup Behavior

For every enabled entry, `Configure`:

1. Decodes the entry into `Config`.
2. Loads AWS configuration for its region.
3. Applies its custom endpoint, if present.
4. Creates an S3 client.
5. Calls `ListBuckets` to verify connectivity.
6. Stores the client under the configuration key.

Consequences:

- Disabled entries are skipped without a network request.
- Invalid decoded entries are skipped.
- Startup validation requires permission for `s3:ListAllMyBuckets`.
- An AWS configuration or connectivity error calls `OnError` and stops processing the remaining entries.
- A nil `ConfigureInput` is allowed; errors are then logged with `slog`.
