package uback

import (
	"reflect"
	"testing"
)

type splitOptionsTest struct {
	s      string
	result [][2]string
}

func TestSplitOptions(t *testing.T) {
	tests := []splitOptionsTest{
		{s: "", result: [][2]string{}},
		{s: "a", result: [][2]string{{"A", "true"}}},
		{s: "a=1", result: [][2]string{{"A", "1"}}},
		{s: "a=1,b=2,c=3", result: [][2]string{{"A", "1"}, {"B", "2"}, {"C", "3"}}},
		{s: "a=1,@b=2,c=3", result: [][2]string{{"A", "1"}, {"@B", "2"}, {"C", "3"}}},
		{s: "a=1,@b=2,c=3,@b=4", result: [][2]string{{"A", "1"}, {"@B", "2"}, {"C", "3"}, {"@B", "4"}}},
		{s: "a=1,b,c=3", result: [][2]string{{"A", "1"}, {"B", "true"}, {"C", "3"}}},
		{s: "a=1\\,b=2,c=3", result: [][2]string{{"A", "1,b=2"}, {"C", "3"}}},
		{s: "a=1\\\\\\,b=2,c=3", result: [][2]string{{"A", "1\\,b=2"}, {"C", "3"}}},
		{s: "a=1\\\\\\\\\\,b=2,c=3", result: [][2]string{{"A", "1\\\\,b=2"}, {"C", "3"}}},
		{s: "a=1\\\\,b=2,c=3", result: [][2]string{{"A", "1\\"}, {"B", "2"}, {"C", "3"}}},
		{s: "a=1\\\\\\\\,b=2,c=3", result: [][2]string{{"A", "1\\\\"}, {"B", "2"}, {"C", "3"}}},
		{s: "a=1\\0,b=2,c=3", result: [][2]string{{"A", "1\\0"}, {"B", "2"}, {"C", "3"}}},
		{s: "a=1,b=2,\\c=3\\", result: [][2]string{{"A", "1"}, {"B", "2"}, {"C", "3\\"}}},
	}

	for _, test := range tests {
		result := SplitOptions(test.s)
		if !reflect.DeepEqual(result, test.result) {
			t.Errorf("does not match: %v %v (from %v)", test.result, result, test.s)
		}
	}
}

func TestEvalOptions(t *testing.T) {
	presets := map[string][]KeyValuePair{
		"src":         {{"KeyFile", "/etc/uback/backup.key"}, {"StateFile", "/var/lib/uback/state/{{.EscapedPath}}.json"}},
		"alt-key":     {{"KeyFile", "/etc/uback/alt.key"}},
		"escape-path": {{"EscapedPath", "{{.Path | replace \"/\" \"-\" | replace \":\" \"-\"}}"}},
		"tar-src":     {{"Preset", "escape-path"}, {"Type", "tar"}, {"Preset", "src"}, {"SnapshotsDir", "/var/lib/uback/tar-snapshots/{{.EscapedPath}}/"}},
	}

	options := []KeyValuePair{
		{"Path", "/etc"},
		{"Preset", "tar-src"},
		{"Preset", "alt-key"},
		{"@RetentionPolicy", "daily=7"},
		{"@RetentionPolicy", "weekly=4"},
	}

	result, err := EvalOptions(options, presets)
	if err != nil {
		t.Error(err)
	}

	expected := &Options{
		String: map[string]string{
			"Type":         "tar",
			"Path":         "/etc",
			"EscapedPath":  "-etc",
			"KeyFile":      "/etc/uback/alt.key",
			"StateFile":    "/var/lib/uback/state/-etc.json",
			"SnapshotsDir": "/var/lib/uback/tar-snapshots/-etc/",
		},
		StrSlice: map[string][]string{
			"RetentionPolicy": {"daily=7", "weekly=4"},
		},
	}

	if !reflect.DeepEqual(expected, result) {
		t.Errorf("result: %v ; expected: %v", result, expected)
	}
}
