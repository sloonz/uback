package uback

import (
	"reflect"
	"testing"
)

type parseRetentionPolicyTest struct {
	s      string
	result RetentionPolicy
}

func TestParseRetentionPolicy(t *testing.T) {
	tests := []parseRetentionPolicyTest{
		{s: "hourly=24", result: RetentionPolicy{Interval: 3600, Count: 24, FullOnly: false}},
		{s: "daily=7", result: RetentionPolicy{Interval: 24 * 3600, Count: 7, FullOnly: false}},
		{s: "weekly=4", result: RetentionPolicy{Interval: 7 * 24 * 3600, Count: 4, FullOnly: false}},
		{s: "monthly=12:full", result: RetentionPolicy{Interval: 30 * 24 * 3600, Count: 12, FullOnly: true}},
		{s: "yearly=2", result: RetentionPolicy{Interval: 365 * 24 * 3600, Count: 2, FullOnly: false}},
		{s: "1h=2", result: RetentionPolicy{Interval: 3600, Count: 2, FullOnly: false}},
		{s: "3d=4", result: RetentionPolicy{Interval: 3 * 24 * 3600, Count: 4, FullOnly: false}},
		{s: "5w=6", result: RetentionPolicy{Interval: 5 * 7 * 24 * 3600, Count: 6, FullOnly: false}},
		{s: "7m=8", result: RetentionPolicy{Interval: 7 * 30 * 24 * 3600, Count: 8, FullOnly: false}},
		{s: "9y=10:full", result: RetentionPolicy{Interval: 9 * 365 * 24 * 3600, Count: 10, FullOnly: true}},
		{s: "11=12", result: RetentionPolicy{Interval: 11, Count: 12, FullOnly: false}},
	}

	for _, test := range tests {
		result, err := ParseRetentionPolicy(test.s)
		if err != nil {
			t.Errorf("failed to parse retention policy: %v: %v", test.s, err)
		} else if !reflect.DeepEqual(result, test.result) {
			t.Errorf("do not match: %v %v (from %v)", test.result, result, test.s)
		}
	}
}

func TestRetentionPolicy(t *testing.T) {
	makeSnapshot := func(s string) *Snapshot {
		sn := Snapshot(s)
		return &sn
	}

	var backups []RetentionPolicySubject
	var policies []RetentionPolicy
	var retained, expectedRetained map[string]struct{}
	var err error

	// Full backups, FullOnly policy
	backups = []RetentionPolicySubject{
		Backup{Snapshot: "20210131T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T120000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T000002.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210128T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 4, FullOnly: true},
	}
	expectedRetained = map[string]struct{}{
		backups[0].Name(): {},
		backups[2].Name(): {},
		backups[3].Name(): {},
		backups[4].Name(): {},
	}

	retained, err = ApplyRetentionPolicies(policies, backups)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(retained, expectedRetained) {
		t.Errorf("expected: %v, got: %v", expectedRetained, retained)
	}

	// Full backups, !FullOnly policy
	backups = []RetentionPolicySubject{
		Backup{Snapshot: "20210131T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T120000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T000002.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210128T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 4, FullOnly: false},
	}
	expectedRetained = map[string]struct{}{
		backups[0].Name(): {},
		backups[2].Name(): {},
		backups[3].Name(): {},
		backups[4].Name(): {},
	}

	retained, err = ApplyRetentionPolicies(policies, backups)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(retained, expectedRetained) {
		t.Errorf("expected: %v, got: %v", expectedRetained, retained)
	}

	// Incremental backups, FullOnly policy
	backups = []RetentionPolicySubject{
		Backup{Snapshot: "20210131T000000.000", BaseSnapshot: makeSnapshot("20210130T120000.000")},
		Backup{Snapshot: "20210130T120000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T000002.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		Backup{Snapshot: "20210129T000001.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		Backup{Snapshot: "20210128T000000.000", BaseSnapshot: makeSnapshot("20210128T000000.000")},
		Backup{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 3, FullOnly: true},
	}
	expectedRetained = map[string]struct{}{
		backups[1].Name(): {},
		backups[5].Name(): {},
	}

	retained, err = ApplyRetentionPolicies(policies, backups)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(retained, expectedRetained) {
		t.Errorf("expected: %v, got: %v", expectedRetained, retained)
	}

	// Incremental backups, !FullOnly policy
	backups = []RetentionPolicySubject{
		Backup{Snapshot: "20210131T000000.000", BaseSnapshot: makeSnapshot("20210130T120000.000")},
		Backup{Snapshot: "20210130T120000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T000002.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		Backup{Snapshot: "20210129T000001.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		Backup{Snapshot: "20210128T000000.000", BaseSnapshot: makeSnapshot("20210128T000000.000")},
		Backup{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 3, FullOnly: false},
	}
	expectedRetained = map[string]struct{}{
		backups[0].Name(): {},
		backups[2].Name(): {},
		backups[3].Name(): {},
	}

	retained, err = ApplyRetentionPolicies(policies, backups)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(retained, expectedRetained) {
		t.Errorf("expected: %v, got: %v", expectedRetained, retained)
	}

	// Merged policies
	backups = []RetentionPolicySubject{
		Backup{Snapshot: "20210131T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T120000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210130T000002.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210128T000000.000", BaseSnapshot: nil},
		Backup{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 3, FullOnly: true},
		{Interval: 2 * 24 * 3600, Count: 3, FullOnly: true},
	}
	expectedRetained = map[string]struct{}{
		backups[0].Name(): {},
		backups[2].Name(): {},
		backups[3].Name(): {},
		backups[5].Name(): {},
	}

	retained, err = ApplyRetentionPolicies(policies, backups)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(retained, expectedRetained) {
		t.Errorf("expected: %v, got: %v", expectedRetained, retained)
	}
}

func TestPrunedBackups(t *testing.T) {
	makeSnapshot := func(s string) *Snapshot {
		sn := Snapshot(s)
		return &sn
	}

	var backups []Backup
	var policies []RetentionPolicy
	var pruned, expectedPruned []Backup
	var err error

	// Keep backups required for a chain
	backups = []Backup{
		{Snapshot: "20210131T000000.000", BaseSnapshot: makeSnapshot("20210130T120000.000")},
		{Snapshot: "20210130T120000.000", BaseSnapshot: makeSnapshot("20210130T000002.000")},
		{Snapshot: "20210130T000002.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		{Snapshot: "20210128T000000.000", BaseSnapshot: makeSnapshot("20210127T000000.000")},
		{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 1, FullOnly: false},
	}
	expectedPruned = []Backup{backups[4], backups[5]}

	pruned, err = GetPrunedBackups(backups, policies)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(pruned, expectedPruned) {
		t.Errorf("expected: %v, got: %v", expectedPruned, pruned)
	}

	// Orphan chains are always pruned
	backups = []Backup{
		{Snapshot: "20210331T000000.000", BaseSnapshot: makeSnapshot("20210330T000000.000")},
		{Snapshot: "20210330T000000.000", BaseSnapshot: makeSnapshot("20210329T000000.000")},
		{Snapshot: "20210329T000000.000", BaseSnapshot: makeSnapshot("20210328T000000.000")},
		{Snapshot: "20210131T000000.000", BaseSnapshot: makeSnapshot("20210130T120000.000")},
		{Snapshot: "20210130T120000.000", BaseSnapshot: makeSnapshot("20210130T000002.000")},
		{Snapshot: "20210130T000002.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		{Snapshot: "20210128T000000.000", BaseSnapshot: makeSnapshot("20210127T000000.000")},
		{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
		{Snapshot: "20210126T000000.000", BaseSnapshot: makeSnapshot("20210125T000000.000")},
		{Snapshot: "20210125T000000.000", BaseSnapshot: makeSnapshot("20210124T000000.000")},
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 1, FullOnly: false},
	}
	expectedPruned = []Backup{backups[0], backups[1], backups[2], backups[7], backups[8], backups[9], backups[10]}

	pruned, err = GetPrunedBackups(backups, policies)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(pruned, expectedPruned) {
		t.Errorf("expected: %v, got: %v", expectedPruned, pruned)
	}

	// Empty policy retains everything (even orphans)
	backups = []Backup{
		{Snapshot: "20210131T000000.000", BaseSnapshot: makeSnapshot("20210130T120000.000")},
		{Snapshot: "20210130T120000.000", BaseSnapshot: makeSnapshot("20210130T000002.000")},
		{Snapshot: "20210130T000002.000", BaseSnapshot: makeSnapshot("20210129T000001.000")},
		{Snapshot: "20210129T000001.000", BaseSnapshot: nil},
		{Snapshot: "20210128T000000.000", BaseSnapshot: makeSnapshot("20210127T000000.000")},
		{Snapshot: "20210127T000000.000", BaseSnapshot: nil},
	}
	policies = nil
	expectedPruned = nil

	pruned, err = GetPrunedBackups(backups, policies)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(pruned, expectedPruned) {
		t.Errorf("expected: %v, got: %v", expectedPruned, pruned)
	}
}

func TestPruneArchives(t *testing.T) {
	var archives []Snapshot
	var policies []RetentionPolicy
	var prunedArchives, prunedBookmarks, expectedPrunedArchives []Snapshot
	var err error

	// Normal policy
	archives = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 2, FullOnly: false},
	}
	expectedPrunedArchives = archives[3:]

	prunedArchives, prunedBookmarks, err = GetPrunedSnapshots(archives, nil, policies, map[string]string{"test-dest": "20210130T120000.000"})
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(prunedArchives, expectedPrunedArchives) {
		t.Errorf("expected: %v, got: %v", expectedPrunedArchives, prunedArchives)
	}

	if len(prunedBookmarks) != 0 {
		t.Errorf("Expected 0 pruned bookmarks")
	}

	// Default policy: prune everything
	archives = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	policies = nil
	expectedPrunedArchives = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}

	prunedArchives, prunedBookmarks, err = GetPrunedSnapshots(archives, nil, policies, map[string]string{"test-dest": "20210130T000002.000"})
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(prunedArchives, expectedPrunedArchives) {
		t.Errorf("expected: %v, got: %v", expectedPrunedArchives, prunedArchives)
	}

	if len(prunedBookmarks) != 0 {
		t.Errorf("Expected 0 pruned bookmarks")
	}
}

func TestPruneBookmarks(t *testing.T) {
	var bookmarks []Snapshot
	var policies []RetentionPolicy
	var prunedArchives, prunedBookmarks, expectedPrunedBookmarks []Snapshot
	var err error

	bookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 2, FullOnly: false},
	}
	expectedPrunedBookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}

	prunedArchives, prunedBookmarks, err = GetPrunedSnapshots(nil, bookmarks, policies, map[string]string{"test-dest": "20210130T120000.000"})
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(prunedBookmarks, expectedPrunedBookmarks) {
		t.Errorf("expected: %v, got: %v", expectedPrunedBookmarks, prunedBookmarks)
	}

	if len(prunedArchives) != 0 {
		t.Errorf("Expected 0 pruned archives")
	}
}

func TestPruneSnapshots(t *testing.T) {
	var archives, bookmarks []Snapshot
	var policies []RetentionPolicy
	var prunedArchives, prunedBookmarks, expectedPrunedArchives, expectedPrunedBookmarks []Snapshot
	var err error

	// Normal policy
	archives = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	bookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	policies = []RetentionPolicy{
		{Interval: 24 * 3600, Count: 2, FullOnly: false},
	}
	expectedPrunedArchives = []Snapshot{
		"20210130T120000.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	expectedPrunedBookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}

	prunedArchives, prunedBookmarks, err = GetPrunedSnapshots(archives, bookmarks, policies, map[string]string{"test-dest": "20210130T120000.000"})
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(prunedArchives, expectedPrunedArchives) {
		t.Errorf("expected: %v, got: %v", expectedPrunedArchives, prunedArchives)
	} else if !reflect.DeepEqual(prunedBookmarks, expectedPrunedBookmarks) {
		t.Errorf("expected: %v, got: %v", expectedPrunedBookmarks, prunedBookmarks)
	}

	// Default policy: prune everything
	archives = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	bookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210130T000002.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}
	policies = nil
	expectedPrunedArchives = archives
	expectedPrunedBookmarks = []Snapshot{
		"20210131T000000.000",
		"20210130T120000.000",
		"20210129T000001.000",
		"20210128T000000.000",
		"20210127T000000.000",
	}

	prunedArchives, prunedBookmarks, err = GetPrunedSnapshots(archives, bookmarks, policies, map[string]string{"test-dest": "20210130T000002.000"})
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(prunedArchives, expectedPrunedArchives) {
		t.Errorf("expected: %v, got: %v", expectedPrunedArchives, prunedArchives)
	} else if !reflect.DeepEqual(prunedBookmarks, expectedPrunedBookmarks) {
		t.Errorf("expected: %v, got: %v", expectedPrunedBookmarks, prunedBookmarks)
	}
}
