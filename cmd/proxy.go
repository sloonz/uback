package cmd

import (
	"github.com/sloonz/uback/destinations"
	"github.com/sloonz/uback/lib"
	"github.com/sloonz/uback/sources"

	"io"
	"net/rpc"
	"os"

	"github.com/hashicorp/yamux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Destination struct {
	dataStream *yamux.Stream
}

func (d *Destination) ListBackups(args *destinations.ListBackupsArgs, reply *[]uback.Backup) error {
	var err error

	dstOpts := newOptionsBuilder(&args.Options, nil).WithDestination()
	if dstOpts.Error != nil {
		return dstOpts.Error
	}

	*reply, err = dstOpts.Destination.ListBackups()
	if err != nil {
		return err
	}

	return nil
}

func (d *Destination) RemoveBackup(args *destinations.RemoveBackupArgs, reply *struct{}) error {
	dstOpts := newOptionsBuilder(&args.Options, nil).WithDestination()
	if dstOpts.Error != nil {
		return dstOpts.Error
	}

	return dstOpts.Destination.RemoveBackup(args.Backup)
}

func (d *Destination) SendBackup(args *destinations.SendBackupArgs, reply *struct{}) error {
	dstOpts := newOptionsBuilder(&args.Options, nil).WithDestination()
	if dstOpts.Error != nil {
		return dstOpts.Error
	}

	if err := dstOpts.Destination.SendBackup(args.Backup, d.dataStream); err != nil {
		return err
	}

	return d.dataStream.Close()
}

func (d *Destination) ReceiveBackup(args *destinations.ReceiveBackupArgs, reply *struct{}) error {
	dstOpts := newOptionsBuilder(&args.Options, nil).WithDestination()
	if dstOpts.Error != nil {
		return dstOpts.Error
	}

	r, err := dstOpts.Destination.ReceiveBackup(args.Backup)
	if err != nil {
		return err
	}

	if _, err = io.Copy(d.dataStream, r); err != nil {
		return err
	}

	if err = r.Close(); err != nil {
		return err
	}

	return d.dataStream.Close()
}

type Source struct {
	dataStream *yamux.Stream
	backup     io.ReadCloser
}

func (s *Source) ListSnapshots(args *sources.ListSnapshotsArgs, reply *[]uback.Snapshot) error {
	var err error

	srcOpts := newOptionsBuilder(&args.Options, nil).WithSource()
	if srcOpts.Error != nil {
		return srcOpts.Error
	}

	*reply, err = srcOpts.Source.ListSnapshots()
	if err != nil {
		return err
	}
	return nil
}

func (s *Source) RemoveSnapshot(args *sources.RemoveSnapshotArgs, reply *struct{}) error {
	srcOpts := newOptionsBuilder(&args.Options, nil).WithSource()
	if srcOpts.Error != nil {
		return srcOpts.Error
	}

	return srcOpts.Source.RemoveSnapshot(args.Snapshot)
}

func (s *Source) CreateBackup(args *sources.CreateBackupArgs, reply *uback.Backup) error {
	var err error

	srcOpts := newOptionsBuilder(&args.Options, nil).WithSource()
	if srcOpts.Error != nil {
		return srcOpts.Error
	}

	*reply, s.backup, err = srcOpts.Source.CreateBackup(args.Snapshot)
	return err
}

func (s *Source) TransmitBackup(args *struct{}, reply *struct{}) error {
	if _, err := io.Copy(s.dataStream, s.backup); err != nil {
		return err
	}

	if err := s.backup.Close(); err != nil {
		return err
	}

	return s.dataStream.Close()
}

var (
	cmdProxy = &cobra.Command{
		Use:    "proxy",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			rwc := uback.ReadWriteCloser{
				ReadCloser:  os.Stdin,
				WriteCloser: os.Stdout,
			}

			session, err := yamux.Server(&rwc, nil)
			if err != nil {
				logrus.Fatalf("Failed to start proxy server: %v", err)
			}
			defer session.Close()

			rpcStream, err := session.AcceptStream()
			if err != nil {
				logrus.Fatalf("Failed to start proxy server: %v", err)
			}

			dataStream, err := session.AcceptStream()
			if err != nil {
				logrus.Fatalf("Failed to start proxy server: %v", err)
			}

			if err = rpc.Register(&Destination{dataStream: dataStream}); err != nil {
				logrus.Fatalf("Failed to start proxy server: %v", err)
			}

			if err = rpc.Register(&Source{dataStream: dataStream}); err != nil {
				logrus.Fatalf("Failed to start proxy server: %v", err)
			}

			rpc.ServeConn(rpcStream)
		},
	}
)
