package cmd

import (
	"github.com/sloonz/uback/destinations"
	"github.com/sloonz/uback/lib"
	"github.com/sloonz/uback/sources"

	"fmt"
	"os"
	"path"

	"filippo.io/age"
	"github.com/sirupsen/logrus"
)

type optionsBuilder struct {
	Options           *uback.Options
	Source            uback.Source
	SourceType        string
	Destination       uback.Destination
	RetentionPolicies []uback.RetentionPolicy
	Identities        []age.Identity
	Recipients        []age.Recipient
	Error             error
}

func newOptionsBuilder(options *uback.Options, err error) *optionsBuilder {
	return &optionsBuilder{Options: options, Error: err}
}

func (o *optionsBuilder) WithSource() *optionsBuilder {
	if o.Error == nil {
		o.Source, o.SourceType, o.Error = sources.New(o.Options)
	}
	return o
}

func (o *optionsBuilder) WithDestination() *optionsBuilder {
	if o.Error == nil {
		o.Destination, o.Error = destinations.New(o.Options)
	}
	return o
}

func (o *optionsBuilder) WithRecipients() *optionsBuilder {
	if o.Error == nil {
		if o.Options.String["NoEncryption"] != "" {
			o.Recipients = nil
		} else {
			o.Recipients, o.Error = uback.LoadRecipients(o.Options.String["KeyFile"], o.Options.String["Key"])
		}
	}
	return o
}

func (o *optionsBuilder) WithIdentities() *optionsBuilder {
	if o.Error == nil {
		if o.Options.String["NoEncryption"] != "" {
			o.Identities = nil
		} else {
			o.Identities, o.Error = uback.LoadIdentities(o.Options.String["KeyFile"], o.Options.String["Key"])
		}
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
