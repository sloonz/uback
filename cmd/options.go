package cmd

import (
	"github.com/sloonz/uback/destinations"
	"github.com/sloonz/uback/lib"
	"github.com/sloonz/uback/sources"
	"github.com/sloonz/uback/x25519"

	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

type optionsBuilder struct {
	Options           *uback.Options
	Source            uback.Source
	Destination       uback.Destination
	RetentionPolicies []uback.RetentionPolicy
	PrivateKey        x25519.PrivateKey
	PublicKey         x25519.PublicKey
	Error             error
}

func newOptionsBuilder(options *uback.Options, err error) *optionsBuilder {
	return &optionsBuilder{Options: options, Error: err}
}

func (o *optionsBuilder) WithSource() *optionsBuilder {
	if o.Error == nil {
		o.Source, o.Error = sources.New(o.Options)
	}
	return o
}

func (o *optionsBuilder) WithDestination() *optionsBuilder {
	if o.Error == nil {
		o.Destination, o.Error = destinations.New(o.Options)
	}
	return o
}

func (o *optionsBuilder) WithPublicKey() *optionsBuilder {
	if o.Error == nil {
		o.PublicKey, o.Error = x25519.LoadPublicKey(o.Options.String["KeyFile"], o.Options.String["Key"])
	}
	return o
}

func (o *optionsBuilder) WithPrivateKey() *optionsBuilder {
	if o.Error == nil {
		o.PrivateKey, o.Error = x25519.LoadPrivateKey(o.Options.String["KeyFile"], o.Options.String["Key"])
	}
	return o
}

func (o *optionsBuilder) WithStringOption(k string) *optionsBuilder {
	if o.Error == nil {
		v := o.Options.String[k]
		if v == "" {
			o.Error = fmt.Errorf("missing option: %s", k)
		}
	}
	return o
}

func (o *optionsBuilder) WithStateFile() *optionsBuilder {
	if _, ok := o.Options.String["StateFile"]; o.Error == nil && ok {
		o.Error = os.MkdirAll(path.Dir(o.Options.String["StateFile"]), 0777)
	}
	return o
}

func (o *optionsBuilder) WithRetentionPolicies() *optionsBuilder {
	if o.Error == nil {
		o.RetentionPolicies, o.Error = o.Options.GetRetentionPolicies()
	}
	return o
}

func (o *optionsBuilder) FatalOnError() *optionsBuilder {
	if o.Error != nil {
		logrus.Fatal(o.Error)
	}
	return o
}
