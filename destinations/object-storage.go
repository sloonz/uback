package destinations

import (
	"github.com/sloonz/uback/lib"

	"context"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

var (
	osLog = logrus.WithFields(logrus.Fields{
		"destination": "object-storage",
	})
)

type objectStorageDestination struct {
	options *uback.Options
	prefix  string
	bucket  string
	client  *minio.Client
}

func newObjectStorageDestination(options *uback.Options) (uback.Destination, error) {
	u, err := url.Parse(options.String["Url"])
	if err != nil {
		osLog.Warnf("cannot parse url: %v", err)
	}

	endpoint := u.Host
	secure := !(u.Scheme == "http")
	accessKeyID := u.User.Username()
	secretAccessKey, _ := u.User.Password()
	bucket := u.Path

	if options.String["Secure"] != "" {
		s, err := strconv.ParseBool(options.String["Secure"])
		if err != nil {
			osLog.Warnf("cannot parse secure option: %v", err)
			secure = true
		} else {
			secure = s
		}
	}

	prefix := strings.Trim(options.String["Prefix"], "/") + "/"
	if prefix == "/" {
		prefix = ""
	}

	if options.String["Endpoint"] != "" {
		endpoint = options.String["Endpoint"]
	}

	if options.String["AccessKeyID"] != "" {
		accessKeyID = options.String["AccessKeyID"]
	}

	if options.String["SecretAccessKey"] != "" {
		secretAccessKey = options.String["SecretAccessKey"]
	}

	if options.String["Bucket"] != "" {
		bucket = options.String["Bucket"]
	}
	bucket = strings.Trim(bucket, "/")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: secure,
	})

	if err != nil {
		return nil, err
	}

	return &objectStorageDestination{options: options, client: client, prefix: prefix, bucket: bucket}, nil
}

func (d *objectStorageDestination) ListBackups() ([]uback.Backup, error) {
	var res []uback.Backup

	ctx, cancel := context.WithCancel(context.Background())
	objectsCh := d.client.ListObjects(ctx, d.bucket, minio.ListObjectsOptions{
		Prefix:    d.prefix,
		Recursive: true,
	})
	defer cancel()

	for obj := range objectsCh {
		if obj.Err != nil {
			return nil, obj.Err
		}

		if strings.HasPrefix(obj.Key, ".") || strings.HasPrefix(obj.Key, "_") || strings.HasSuffix(obj.Key, "/") {
			continue
		}

		backup, err := uback.ParseBackupFilename(path.Base(obj.Key), true)
		if err != nil {
			osLog.WithFields(logrus.Fields{
				"key": obj.Key,
			})
			logrus.Warnf("invalid backup file: %v", err)
			continue
		}

		res = append(res, backup)
	}

	return res, nil
}

func (d *objectStorageDestination) RemoveBackup(backup uback.Backup) error {
	return d.client.RemoveObject(context.Background(), d.bucket, d.prefix+backup.Filename(), minio.RemoveObjectOptions{})
}

func (d *objectStorageDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	osLog.Printf("writing backup to %s", d.prefix+backup.Filename())
	_, err := d.client.PutObject(context.Background(), d.bucket, d.prefix+backup.Filename(), data, -1, minio.PutObjectOptions{})
	if err != nil {
		d.client.RemoveObject(context.Background(), d.bucket, d.prefix+backup.Filename(), minio.RemoveObjectOptions{}) //nolint:errcheck
	}
	return err
}

func (d *objectStorageDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	return d.client.GetObject(context.Background(), d.bucket, d.prefix+backup.Filename(), minio.GetObjectOptions{})
}
