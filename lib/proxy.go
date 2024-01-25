package uback

import (
	"io"
	"net/rpc"
	"strings"

	"github.com/hashicorp/yamux"
	"github.com/sirupsen/logrus"
)

type ReadWriteCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rwc *ReadWriteCloser) Close() error {
	if err := rwc.ReadCloser.Close(); err != nil {
		_ = rwc.WriteCloser.Close()
		return err
	}
	return rwc.WriteCloser.Close()
}

func OpenProxy(logger *logrus.Entry, command []string) (*yamux.Session, *rpc.Client, *yamux.Stream, error) {
	cmd := BuildCommand(command)
	cmd.Stdout = nil
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	rwc := ReadWriteCloser{
		ReadCloser:  stdout,
		WriteCloser: stdin,
	}

	err = StartCommand(logger, cmd)
	if err != nil {
		return nil, nil, nil, err
	}

	session, err := yamux.Client(&rwc, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	rpcStream, err := session.OpenStream()
	if err != nil {
		return nil, nil, nil, err
	}

	dataStream, err := session.OpenStream()
	if err != nil {
		return nil, nil, nil, err
	}

	rpcClient := rpc.NewClient(rpcStream)
	return session, rpcClient, dataStream, nil
}

func CloseProxy(session *yamux.Session, rpcClient *rpc.Client) error {
	err := rpcClient.Close()
	if err != nil {
		_ = session.Close()
		return err
	}
	return session.Close()
}

func ProxiedOptions(options *Options) Options {
	opts := Options{
		String:   make(map[string]string),
		StrSlice: make(map[string][]string),
	}

	for k, v := range options.String {
		if k != "Proxy" && k != "Command" && k != "Type" {
			opts.String[strings.TrimPrefix(k, "Proxy")] = v
		}
	}

	for k, v := range options.StrSlice {
		if k != "Proxy" && k != "Command" && k != "Type" {
			opts.StrSlice[strings.TrimPrefix(k, "Proxy")] = v
		}
	}

	return opts
}
