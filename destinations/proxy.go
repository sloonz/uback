package destinations

import (
	"github.com/sloonz/uback/lib"

	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	ErrProxyCommandMissing = errors.New("proxy destination: missing command")
	proxyLog               = logrus.WithFields(logrus.Fields{
		"destination": "proxy",
	})
)

type ListBackupsArgs struct {
	uback.Options
}

type RemoveBackupArgs struct {
	uback.Options
	uback.Backup
}

type SendBackupArgs struct {
	uback.Options
	uback.Backup
}

type ReceiveBackupArgs struct {
	uback.Options
	uback.Backup
}

type proxyDestination struct {
	options *uback.Options
	command []string
}

func newProxyDestination(options *uback.Options) (uback.Destination, error) {
	command := options.GetCommand("Command", nil)
	if len(command) == 0 {
		return nil, ErrProxyCommandMissing
	}

	return &proxyDestination{options: options, command: command}, nil
}

func (d *proxyDestination) ListBackups() ([]uback.Backup, error) {
	session, rpcClient, _, err := uback.OpenProxy(proxyLog, d.command)
	if err != nil {
		return nil, fmt.Errorf("Failed to open proxy session: %v", err)
	}
	defer uback.CloseProxy(session, rpcClient) //nolint: errcheck

	var backups []uback.Backup
	err = rpcClient.Call("Destination.ListBackups", &ListBackupsArgs{Options: uback.ProxiedOptions(d.options)}, &backups)
	if err != nil {
		return nil, err
	}

	err = uback.CloseProxy(session, rpcClient)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

func (d *proxyDestination) RemoveBackup(backup uback.Backup) error {
	session, rpcClient, _, err := uback.OpenProxy(proxyLog, d.command)
	if err != nil {
		return fmt.Errorf("Failed to open proxy session: %v", err)
	}
	defer uback.CloseProxy(session, rpcClient) //nolint: errcheck

	return rpcClient.Call("Destination.RemoveBackup", &RemoveBackupArgs{Options: uback.ProxiedOptions(d.options), Backup: backup}, nil)
}

func (d *proxyDestination) SendBackup(backup uback.Backup, data io.Reader) error {
	session, rpcClient, dataStream, err := uback.OpenProxy(proxyLog, d.command)
	if err != nil {
		return fmt.Errorf("Failed to open proxy session: %v", err)
	}
	defer uback.CloseProxy(session, rpcClient) //nolint: errcheck

	call := rpcClient.Go("Destination.SendBackup", &SendBackupArgs{Options: uback.ProxiedOptions(d.options), Backup: backup}, nil, nil)

	if _, err := io.Copy(dataStream, data); err != nil {
		return err
	}

	if err := dataStream.Close(); err != nil {
		return err
	}

	<-call.Done

	return call.Error
}

func (d *proxyDestination) ReceiveBackup(backup uback.Backup) (io.ReadCloser, error) {
	session, rpcClient, dataStream, err := uback.OpenProxy(proxyLog, d.command)
	if err != nil {
		return nil, fmt.Errorf("Failed to open proxy session: %v", err)
	}

	pr, pw := io.Pipe()
	call := rpcClient.Go("Destination.ReceiveBackup", &ReceiveBackupArgs{Options: uback.ProxiedOptions(d.options), Backup: backup}, nil, nil)

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

	return pr, nil
}
