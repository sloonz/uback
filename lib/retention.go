package uback

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var intervalAliases = map[string]string{
	"yearly":  "1y",
	"monthly": "1m",
	"weekly":  "1w",
	"daily":   "1d",
	"hourly":  "1h",
}

type RetentionPolicy struct {
	Interval int  // Minimum interval between two backups
	Count    int  // Maximum number of retained backups
	FullOnly bool // If true, only retain full backups
}

// Can be a backup or a snapshot
type RetentionPolicySubject interface {
	Time() (time.Time, error)
	IsFull() bool
	Name() string
}

// Parse an interval. Can be expressed in hours, days, weeks, months or years.
// Return the time interval in seconds.
func ParseInterval(intv string) (int, error) {
	alias, ok := intervalAliases[intv]
	if ok {
		intv = alias
	}

	if len(intv) == 0 {
		return 0, fmt.Errorf("empty interval")
	}

	var result int
	var suffix byte
	var err error
	if strings.Contains("ymwdh", string(intv[len(intv)-1])) {
		result, err = strconv.Atoi(intv[:len(intv)-1])
		suffix = intv[len(intv)-1]
	} else {
		result, err = strconv.Atoi(intv)
	}
	if err != nil {
		return 0, err
	}

	switch suffix {
	case 'y':
		result *= 365 * 24 * 3600
	case 'm':
		result *= 30 * 24 * 3600
	case 'w':
		result *= 7 * 24 * 3600
	case 'd':
		result *= 24 * 3600
	case 'h':
		result *= 3600
	default:
		if suffix != 0 {
			return 0, fmt.Errorf("invalid suffix: %v", suffix)
		}
	}

	return result, nil
}

func ParseRetentionPolicy(policy string) (RetentionPolicy, error) {
	kv := strings.SplitN(policy, "=", 2)
	if len(kv) != 2 {
		return RetentionPolicy{}, fmt.Errorf("invalid item")
	}

	k := strings.TrimSpace(kv[0])
	v := strings.Split(kv[1], ":")
	count, err := strconv.Atoi(strings.TrimSpace(v[0]))
	if err != nil {
		return RetentionPolicy{}, err
	}

	intv, err := ParseInterval(k)
	if err != nil {
		return RetentionPolicy{}, err
	}

	parsedPolicy := RetentionPolicy{
		Interval: intv,
		Count:    count,
		FullOnly: false,
	}

	if len(v) > 1 {
		for _, opt := range v[1:] {
			if strings.TrimSpace(opt) == "full" {
				parsedPolicy.FullOnly = true
			} else {
				return RetentionPolicy{}, fmt.Errorf("invalid option")
			}
		}
	}

	return parsedPolicy, nil
}

// Apply retention policies to a set of subjects, returning a set of retained subject names
func ApplyRetentionPolicies(policies []RetentionPolicy, subjects []RetentionPolicySubject) (map[string]struct{}, error) {
	retained := make(map[string]struct{})
	for _, policy := range policies {
		var lastRetainedTime time.Time
		retainedCount := 0
		for _, subject := range subjects {
			if retainedCount >= policy.Count {
				break
			}
			if policy.FullOnly && !subject.IsFull() {
				continue
			}
			t, err := subject.Time()
			if err != nil {
				return nil, err
			}
			if retainedCount == 0 || lastRetainedTime.Sub(t).Seconds() >= 0.9*float64(policy.Interval) {
				lastRetainedTime = t
				retained[subject.Name()] = struct{}{}
				retainedCount++
			}
		}
	}
	return retained, nil
}

// Get backups from a destination not retained by a given retention policy
func GetPrunedBackups(backups []Backup, policies []RetentionPolicy) ([]Backup, error) {
	index := MakeIndex(backups)
	chains := make(map[string][]Backup)

	subjects := make([]RetentionPolicySubject, 0, len(backups))
	for _, b := range backups {
		chain, ok := GetFullChain(b, index)
		if ok {
			subjects = append(subjects, b)
			chains[b.Name()] = chain
		}
	}

	if len(policies) == 0 {
		// Default policy for backups is to retain everything
		logrus.Warn("no retention policies set for destination, keeping everything")
		return nil, nil
	}

	retained, err := ApplyRetentionPolicies(policies, subjects)
	if err != nil {
		return nil, err
	}

	retainedChainFronts := make([]string, 0, len(retained))
	for r := range retained {
		retainedChainFronts = append(retainedChainFronts, r)
	}
	for _, r := range retainedChainFronts {
		for _, b := range chains[r] {
			retained[b.Name()] = struct{}{}
		}
	}

	pruned := make([]Backup, 0, len(backups)-len(retained))
	for _, b := range backups {
		if _, ok := retained[b.Name()]; !ok {
			pruned = append(pruned, b)
		}
	}

	return pruned, nil
}

// Get snapshots from a source not retained by a given retention policy
func GetPrunedSnapshots(snapshots []Snapshot, policies []RetentionPolicy, state map[string]string) ([]Snapshot, error) {
	subjects := make([]RetentionPolicySubject, 0, len(snapshots))
	for _, s := range snapshots {
		subjects = append(subjects, s)
	}

	var retained map[string]struct{}
	if len(policies) == 0 {
		// Default policy for snapshots is to retain nothing
		retained = make(map[string]struct{})
	} else {
		var err error
		retained, err = ApplyRetentionPolicies(policies, subjects)
		if err != nil {
			return nil, err
		}
	}

	// Retain snapshots used by destinations
	for _, s := range state {
		retained[s] = struct{}{}
	}

	pruned := make([]Snapshot, 0, len(snapshots)-len(retained))
	for _, s := range snapshots {
		if _, ok := retained[s.Name()]; !ok {
			pruned = append(pruned, s)
		}
	}

	return pruned, nil
}

// Prune backups from a destinations according to a retention policy
func PruneBackups(dst Destination, backups []Backup, policies []RetentionPolicy) error {
	prunedBackups, err := GetPrunedBackups(backups, policies)
	if err != nil {
		return err
	}

	for _, b := range prunedBackups {
		log := logrus.WithFields(logrus.Fields{"backup": string(b.Snapshot)})
		log.Printf("removing backup")
		err = dst.RemoveBackup(b)
		if err != nil {
			log.Warnf("cannot prune backup: %v", err)
		}
	}

	return nil
}

// Prune snapshots from a source accoruding to a retention policy
func PruneSnapshots(src Source, snapshots []Snapshot, policies []RetentionPolicy, state map[string]string) error {
	prunedSnapshots, err := GetPrunedSnapshots(snapshots, policies, state)
	if err != nil {
		return err
	}

	for _, s := range prunedSnapshots {
		log := logrus.WithFields(logrus.Fields{"snapshot": string(s)})
		log.Printf("deleting snapshot")
		err = src.RemoveSnapshot(s)
		if err != nil {
			log.Warnf("cannot prune snapshot: %v", err)
		}
	}

	return nil
}
