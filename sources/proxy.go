package sources

import (
	uback "github.com/sloonz/uback/lib"

	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	ErrProxyCommandMissing = errors.New("proxy source: missing command")
	ErrProxyMissingType    = errors.New("proxy source: missing proxy-type")
	ErrProxyNoRestoration  = errors.New("proxy source: restoration not implemented")
	proxyLog               = logrus.WithFields(logrus.Fields{
		"source": "proxy",
	})
)

type ListSnapshotsArgs struct {
	uback.Options
}

type RemoveSnapshotArgs struct {
	uback.Options
	uback.Snapshot
}

type CreateBackupArgs struct {
	uback.Options
	*uback.Snapshot
}

type proxySource struct {
	options *uback.Options
	command []string
}

func newProxySource(options *uback.Options) (uback.Source, string, error) {
	command := options.GetCommand("Command", nil)
	if len(command) == 0 {
		return nil, "", ErrProxyCommandMissing
	}

	typ := "proxy"
	prefix := "Proxy"
	for typ == "proxy" {
		var ok bool
		typ, ok = options.String[prefix+"Type"]
		if !ok {
			return nil, "", ErrProxyMissingType
		} else {
			prefix += "Proxy"
		}
	}

	return &proxySource{options: options, command: command}, typ, nil
}

func (s *proxySource) listSnapshots(kind string) ([]uback.Snapshot, error) {
	session, rpcClient, _, err := uback.OpenProxy(proxyLog, s.command)
	if err != nil {
		return nil, fmt.Errorf("Failed to open proxy session: %v", err)
	}
	defer uback.CloseProxy(session, rpcClient) //nolint: errcheck

	var snapshots []uback.Snapshot
	err = rpcClient.Call("Source.List"+kind, &ListSnapshotsArgs{Options: uback.ProxiedOptions(s.options)}, &snapshots)
	if err != nil {
		return nil, err
	}

	err = uback.CloseProxy(session, rpcClient)
	if err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Part of uback.Source interface
func (s *proxySource) ListArchives() ([]uback.Snapshot, error) {
	return s.listSnapshots("Archives")
}

// Part of uback.Source interface
func (s *proxySource) ListBookmarks() ([]uback.Snapshot, error) {
	return s.listSnapshots("Bookmarks")
}

// Part of uback.Source interface
func (s *proxySource) removeSnapshot(kind string, snapshot uback.Snapshot) error {
	session, rpcClient, _, err := uback.OpenProxy(proxyLog, s.command)
	if err != nil {
		return fmt.Errorf("Failed to open proxy session: %v", err)
	}
	defer uback.CloseProxy(session, rpcClient) //nolint: errcheck

	return rpcClient.Call("Source.Remove"+kind, &RemoveSnapshotArgs{Options: uback.ProxiedOptions(s.options), Snapshot: snapshot}, nil)
}

// Part of uback.Source interface
func (s *proxySource) RemoveArchive(snapshot uback.Snapshot) error {
	return s.removeSnapshot("Archive", snapshot)
}

// Part of uback.Source interface
func (s *proxySource) RemoveBookmark(snapshot uback.Snapshot) error {
	return s.removeSnapshot("Bookmark", snapshot)
}

// Part of uback.Source interface
func (s *proxySource) CreateBackup(baseSnapshot *uback.Snapshot) (uback.Backup, io.ReadCloser, error) {
	session, rpcClient, dataStream, err := uback.OpenProxy(proxyLog, s.command)
	if err != nil {
		return uback.Backup{}, nil, fmt.Errorf("Failed to open proxy session: %v", err)
	}

	var backup uback.Backup
	err = rpcClient.Call("Source.CreateBackup", &CreateBackupArgs{Options: uback.ProxiedOptions(s.options), Snapshot: baseSnapshot}, &backup)
	if err != nil {
		return uback.Backup{}, nil, err
	}

	pr, pw := io.Pipe()
	call := rpcClient.Go("Source.TransmitBackup", struct{}{}, nil, nil)

	ch := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-call.Done
		ch <- call.Error
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if _, err := io.Copy(pw, dataStream); err != nil {
			ch <- err
			return
		}

		if err := dataStream.Close(); err != nil {
			ch <- err
			return
		}
	}()

	go func() {
		defer uback.CloseProxy(session, rpcClient) //nolint: errcheck
		wg.Wait()
		close(ch)
		for err := range ch {
			if err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		pw.Close()
	}()

	return backup, pr, nil
}

// Part of uback.Source interface
func (s *proxySource) RestoreBackup(targetDir string, backup uback.Backup, data io.Reader) error {
	return ErrProxyNoRestoration
}
