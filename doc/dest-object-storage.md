# object-storage Destination

Store backups on a S3-compatible object-storage service. It can
be configured either by providing an URL or by giving individual
options. If both an URL and an option are provided, the specific option
takes precedence. Here is a correspondence between URL parts and the
specific options :

| URL part              | Option          |
|-----------------------|-----------------|
| Host (including port) | Endpoint        |
| Username              | AccessKeyID     |
| Password              | SecretAccessKey |
| Scheme                | Secure          |
| Path                  | Bucket          |
| (none)                | Prefix          |

## Limitations

Since S3-compatible object storages do not provide a rename operation,
backups currently uploaded do not have a different naming scheme from
backups whose upload process have been successfully completed. Partially
uploaded backups may therefore appear in the backups list.

## Options

### URL

Allows you to specify multiple options in one. See above for the
correspondence between URL parts and specific options.

### Endpoint

Required, can be provided via URL.

Address of the object storage server (excluding the protocol/scheme,
but including the port if it’s not the standard HTTP/HTTPS port).

### Secure

Optional, defaults to false. Can be provided via URL.

Set to true (or scheme to "http" if provided via the Url option) to
access to the object storage server with plain HTTP, without SSL.

### AccessKeyID

Required, can be provided via URL.

### SecretAccessKey

Required, can be provided via URL.

### Bucket

Required, can be provided via URL.

### Prefix

Optional, defaults to empty.

If you want to host multiple destinations on a single bucket, you have
to use a different prefix for each destination.
