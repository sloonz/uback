package uback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/gobuffalo/flect"
	"github.com/google/shlex"
	"github.com/sirupsen/logrus"
)

var splitOptionsRe = regexp.MustCompile(`(?:[^\\]|^)(?:\\\\)*,`)

type KeyValuePair = [2]string

// Store parsed and evaluated options
type Options struct {
	// All normal (non-"@"-prefixed) options
	String map[string]string

	// All noslice (non-"@"-prefixed) options
	// Keys have their "@" prefix stripped
	StrSlice map[string][]string
}

func (o *Options) merge() map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range o.String {
		res[k] = v
	}
	for k, v := range o.StrSlice {
		res["@"+k] = v
	}
	return res
}

// Get a command, for @Command options
// This supports a shorthand, Command="sudo mycommand" (for example)
// where the simple string will be parsed into a slice following shell syntax
func (o *Options) GetCommand(key string, defaults []string) []string {
	if ss, ok := o.StrSlice[key]; ok {
		return ss
	}

	if s, ok := o.String[key]; ok {
		res, err := shlex.Split(s)
		if err != nil {
			logrus.Warnf("cannot parse %s: %s", key, err)
		} else {
			return res
		}
	}

	return defaults
}

func (o *Options) GetBoolean(key string, defaults bool) (bool, error) {
	if s, ok := o.String[key]; ok {
		ls := strings.ToLower(s)
		switch ls {
		case "1", "true", "yes":
			return true, nil
		case "0", "false", "no":
			return false, nil
		default:
			return false, fmt.Errorf("invalid boolean: %s", s)
		}
	}

	return defaults, nil
}

func (o *Options) GetString(key string, defaults string) string {
	if s, ok := o.String[key]; ok {
		return s
	} else {
		return defaults
	}
}

// Parse retention policies
func (o *Options) GetRetentionPolicies() ([]RetentionPolicy, error) {
	var policies []RetentionPolicy
	for _, p := range o.StrSlice["RetentionPolicy"] {
		parsedPolicy, err := ParseRetentionPolicy(p)
		if err != nil {
			return nil, err
		}
		policies = append(policies, parsedPolicy)
	}

	return policies, nil
}

func parseOption(option string) (string, string) {
	s := strings.SplitN(strings.ReplaceAll(strings.ReplaceAll(option, "\\,", ","), "\\\\", "\\"), "=", 2)
	if len(s) == 0 {
		return "", ""
	}

	var prefix string
	k := s[0]
	if len(k) > 0 && k[0] == '@' {
		prefix = string(k[0])
		k = k[1:]
	}

	if len(s) == 1 {
		return prefix + flect.Pascalize(k), "true"
	} else if len(s) == 2 {
		return prefix + flect.Pascalize(k), s[1]
	} else {
		panic("should not happen")
	}
}

// Split a command line into a list of key-value pairs, separated by a comma
func SplitOptions(options string) []KeyValuePair {
	result := make([]KeyValuePair, 0)
	indices := splitOptionsRe.FindAllStringIndex(options, -1)

	prevPos := 0
	for _, idx := range indices {
		pos := idx[1]
		k, v := parseOption(options[prevPos : pos-1])
		if k != "" {
			result = append(result, KeyValuePair{k, v})
		}
		prevPos = pos
	}

	k, v := parseOption(options[prevPos:])
	if k != "" {
		result = append(result, KeyValuePair{k, v})
	}

	return result
}

// Load presets from a directory
func ReadPresets(presetsDir string) (map[string][]KeyValuePair, error) {
	entries, err := os.ReadDir(presetsDir)
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	presets := make(map[string][]KeyValuePair)
	for _, entry := range entries {
		f, err := os.Open(path.Join(presetsDir, entry.Name()))
		if err != nil {
			logrus.Warn(err)
			continue
		}
		defer f.Close()

		var options []KeyValuePair
		dec := json.NewDecoder(f)
		err = dec.Decode(&options)
		if err != nil {
			logrus.Warn(err)
		}

		name := strings.TrimRight(entry.Name(), ".json")
		presets[name] = options
	}

	return presets, nil
}

func evalOptions(result *Options, kvs []KeyValuePair, presets map[string][]KeyValuePair) error {
	for _, kv := range kvs {
		k, v := kv[0], kv[1]

		tpl, err := template.New(k).Funcs(sprig.TxtFuncMap()).Parse(v)
		if err != nil {
			logrus.Warnf("failed to evaluate %v: %v", k, err)
		} else {
			buf := bytes.NewBuffer(nil)
			err = tpl.Execute(buf, result.merge())
			if err != nil {
				logrus.Warnf("failed to evaluate %v: %v", k, err)
			} else {
				v = buf.String()
			}
		}

		if k == "Preset" {
			presetOptions, ok := presets[v]
			if ok {
				err := evalOptions(result, presetOptions, presets)
				if err != nil {
					return err
				}
			} else {
				logrus.Warnf("preset %s not found", v)
			}
		} else if len(k) > 0 && k[0] == '@' {
			result.StrSlice[k[1:]] = append(result.StrSlice[k[1:]], v)
		} else {
			result.String[k] = v
		}
	}
	return nil
}

// Evaluate raw key-value pairs (evaluate values as a template and substitute presets)
func EvalOptions(kvs []KeyValuePair, presets map[string][]KeyValuePair) (*Options, error) {
	options := &Options{
		String:   make(map[string]string),
		StrSlice: make(map[string][]string),
	}
	err := evalOptions(options, kvs, presets)
	if err != nil {
		return nil, err
	}

	return options, nil
}
