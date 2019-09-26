// Copyright 2019. PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/coreos/go-semver/semver"
	gh "github.com/dustin/go-humanize"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var defaultSchedulers = SchedulerConfigs{
	{Type: "balance-region"},
	{Type: "balance-leader"},
	{Type: "hot-region"},
	{Type: "label"},
}

// Config is the pd server configuration.
type PdConfig struct {
	*flag.FlagSet `json:"-"`

	Version bool `json:"-"`

	ClientUrls          string `toml:"client-urls" json:"client-urls"`
	PeerUrls            string `toml:"peer-urls" json:"peer-urls"`
	AdvertiseClientUrls string `toml:"advertise-client-urls" json:"advertise-client-urls"`
	AdvertisePeerUrls   string `toml:"advertise-peer-urls" json:"advertise-peer-urls"`

	Name    string `toml:"name" json:"name"`
	DataDir string `toml:"data-dir" json:"data-dir"`

	InitialCluster      string `toml:"initial-cluster" json:"initial-cluster"`
	InitialClusterState string `toml:"initial-cluster-state" json:"initial-cluster-state"`

	// Join to an existing pd cluster, a string of endpoints.
	Join string `toml:"join" json:"join"`

	// LeaderLease time, if leader doesn't update its TTL
	// in etcd after lease time, etcd will expire the leader key
	// and other servers can campaign the leader again.
	// Etcd onlys support seoncds TTL, so here is second too.
	LeaderLease int64 `toml:"lease" json:"lease"`

	// Log related config.
	Log log.Config `toml:"log" json:"log"`

	// Backward compatibility.
	LogFileDeprecated  string `toml:"log-file" json:"log-file"`
	LogLevelDeprecated string `toml:"log-level" json:"log-level"`

	// TsoSaveInterval is the interval to save timestamp.
	TsoSaveInterval Duration `toml:"tso-save-interval" json:"tso-save-interval"`

	Metric MetricConfig `toml:"metric" json:"metric"`

	Schedule ScheduleConfig `toml:"schedule" json:"schedule"`

	Replication ReplicationConfig `toml:"replication" json:"replication"`

	Namespace map[string]NamespaceConfig `json:"namespace"`

	ClusterVersion semver.Version `json:"cluster-version"`

	// QuotaBackendBytes Raise alarms when backend size exceeds the given quota. 0 means use the default quota.
	// the default size is 2GB, the maximum is 8GB.
	QuotaBackendBytes ByteSize `toml:"quota-backend-bytes" json:"quota-backend-bytes"`
	// AutoCompactionMode is either 'periodic' or 'revision'. The default value is 'periodic'.
	AutoCompactionMode string `toml:"auto-compaction-mode" json:"auto-compaction-mode"`
	// AutoCompactionRetention is either duration string with time unit
	// (e.g. '5m' for 5-minute), or revision unit (e.g. '5000').
	// If no time unit is provided and compaction mode is 'periodic',
	// the unit defaults to hour. For example, '5' translates into 5-hour.
	// The default retention is 1 hour.
	// Before etcd v3.3.x, the type of retention is int. We add 'v2' suffix to make it backward compatible.
	AutoCompactionRetention string `toml:"auto-compaction-retention" json:"auto-compaction-retention-v2"`

	// TickInterval is the interval for etcd Raft tick.
	TickInterval Duration `toml:"tick-interval"`
	// ElectionInterval is the interval for etcd Raft election.
	ElectionInterval Duration `toml:"election-interval"`
	// Prevote is true to enable Raft Pre-Vote.
	// If enabled, Raft runs an additional election phase
	// to check whether it would get enough votes to win
	// an election, thus minimizing disruptions.
	PreVote bool `toml:"enable-prevote"`

	Security SecurityConfig `toml:"security" json:"security"`

	LabelProperty LabelPropertyConfig `toml:"label-property" json:"label-property"`

	configFile string

	// For all warnings during parsing.
	WarningMsgs []string

	// NamespaceClassifier is for classifying stores/regions into different
	// namespaces.
	NamespaceClassifier string `toml:"namespace-classifier" json:"namespace-classifier"`

	// Only test can change them.
	nextRetryDelay             time.Duration
	disableStrictReconfigCheck bool

	heartbeatStreamBindInterval Duration

	leaderPriorityCheckInterval Duration

	logger   *zap.Logger
	logProps *log.ZapProperties
}

// ScheduleConfig is the schedule configuration.
type ScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	MaxSnapshotCount    uint64 `toml:"max-snapshot-count,omitempty" json:"max-snapshot-count"`
	MaxPendingPeerCount uint64 `toml:"max-pending-peer-count,omitempty" json:"max-pending-peer-count"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	MaxMergeRegionSize uint64 `toml:"max-merge-region-size,omitempty" json:"max-merge-region-size"`
	MaxMergeRegionKeys uint64 `toml:"max-merge-region-keys,omitempty" json:"max-merge-region-keys"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	SplitMergeInterval Duration `toml:"split-merge-interval,omitempty" json:"split-merge-interval"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	PatrolRegionInterval Duration `toml:"patrol-region-interval,omitempty" json:"patrol-region-interval"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	MaxStoreDownTime Duration `toml:"max-store-down-time,omitempty" json:"max-store-down-time"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	LeaderScheduleLimit uint64 `toml:"leader-schedule-limit,omitempty" json:"leader-schedule-limit"`
	// RegionScheduleLimit is the max coexist region schedules.
	RegionScheduleLimit uint64 `toml:"region-schedule-limit,omitempty" json:"region-schedule-limit"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	ReplicaScheduleLimit uint64 `toml:"replica-schedule-limit,omitempty" json:"replica-schedule-limit"`
	// MergeScheduleLimit is the max coexist merge schedules.
	MergeScheduleLimit uint64 `toml:"merge-schedule-limit,omitempty" json:"merge-schedule-limit"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	HotRegionScheduleLimit uint64 `toml:"hot-region-schedule-limit,omitempty" json:"hot-region-schedule-limit"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	HotRegionCacheHitsThreshold uint64 `toml:"hot-region-cache-hits-threshold,omitempty" json:"hot-region-cache-hits-threshold"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	TolerantSizeRatio float64 `toml:"tolerant-size-ratio,omitempty" json:"tolerant-size-ratio"`
	//
	//      high space stage         transition stage           low space stage
	//   |--------------------|-----------------------------|-------------------------|
	//   ^                    ^                             ^                         ^
	//   0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity
	//
	// LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
	// When in low space, store region score increases to very large and varies inversely with available size.
	LowSpaceRatio float64 `toml:"low-space-ratio,omitempty" json:"low-space-ratio"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	HighSpaceRatio float64 `toml:"high-space-ratio,omitempty" json:"high-space-ratio"`
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode
	DisableLearner bool `toml:"disable-raft-learner" json:"disable-raft-learner,string"`

	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	DisableRemoveDownReplica bool `toml:"disable-remove-down-replica" json:"disable-remove-down-replica,string"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// repalcing offline replicas.
	DisableReplaceOfflineReplica bool `toml:"disable-replace-offline-replica" json:"disable-replace-offline-replica,string"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	DisableMakeUpReplica bool `toml:"disable-make-up-replica" json:"disable-make-up-replica,string"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	DisableRemoveExtraReplica bool `toml:"disable-remove-extra-replica" json:"disable-remove-extra-replica,string"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	DisableLocationReplacement bool `toml:"disable-location-replacement" json:"disable-location-replacement,string"`
	// DisableNamespaceRelocation is the option to prevent namespace checker
	// from moving replica to the target namespace.
	DisableNamespaceRelocation bool `toml:"disable-namespace-relocation" json:"disable-namespace-relocation,string"`

	// Schedulers support for loding customized schedulers
	Schedulers SchedulerConfigs `toml:"schedulers,omitempty" json:"schedulers-v2"` // json v2 is for the sake of compatible upgrade
}

// SchedulerConfigs is a slice of customized scheduler configuration.
type SchedulerConfigs []SchedulerConfig

// SchedulerConfig is customized scheduler configuration
type SchedulerConfig struct {
	Type    string   `toml:"type" json:"type"`
	Args    []string `toml:"args,omitempty" json:"args"`
	Disable bool     `toml:"disable" json:"disable"`
}

// ReplicationConfig is the replication configuration.
type ReplicationConfig struct {
	// MaxReplicas is the number of replicas for each region.
	MaxReplicas uint64 `toml:"max-replicas,omitempty" json:"max-replicas"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	LocationLabels StringSlice `toml:"location-labels,omitempty" json:"location-labels"`
}

// NamespaceConfig is to overwrite the global setting for specific namespace
type NamespaceConfig struct {
	// LeaderScheduleLimit is the max coexist leader schedules.
	LeaderScheduleLimit uint64 `json:"leader-schedule-limit"`
	// RegionScheduleLimit is the max coexist region schedules.
	RegionScheduleLimit uint64 `json:"region-schedule-limit"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	ReplicaScheduleLimit uint64 `json:"replica-schedule-limit"`
	// MergeScheduleLimit is the max coexist merge schedules.
	MergeScheduleLimit uint64 `json:"merge-schedule-limit"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	HotRegionScheduleLimit uint64 `json:"hot-region-schedule-limit"`
	// MaxReplicas is the number of replicas for each region.
	MaxReplicas uint64 `json:"max-replicas"`
}

// SecurityConfig is the configuration for supporting tls.
type SecurityConfig struct {
	// CAPath is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	CAPath string `toml:"cacert-path" json:"cacert-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path" json:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path" json:"key-path"`
}

// StoreLabel is the config item of LabelPropertyConfig.
type StoreLabel struct {
	Key   string `toml:"key" json:"key"`
	Value string `toml:"value" json:"value"`
}

// LabelPropertyConfig is the config section to set properties to store labels.
type LabelPropertyConfig map[string][]StoreLabel

// MetricConfig is the metric configuration.
type MetricConfig struct {
	PushJob      string   `toml:"job" json:"job"`
	PushAddress  string   `toml:"address" json:"address"`
	PushInterval Duration `toml:"interval" json:"interval"`
}

// Duration is a wrapper of time.Duration for TOML and JSON.
type Duration struct {
	time.Duration
}

// NewDuration creates a Duration from time.Duration.
func NewDuration(duration time.Duration) Duration {
	return Duration{Duration: duration}
}

// MarshalJSON returns the duration as a JSON string.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// UnmarshalJSON parses a JSON string into the duration.
func (d *Duration) UnmarshalJSON(text []byte) error {
	s, err := strconv.Unquote(string(text))
	if err != nil {
		return errors.WithStack(err)
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return errors.WithStack(err)
	}
	d.Duration = duration
	return nil
}

// UnmarshalText parses a TOML string into the duration.
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return errors.WithStack(err)
}

// ByteSize is a retype uint64 for TOML and JSON.
type ByteSize uint64

// MarshalJSON returns the size as a JSON string.
func (b ByteSize) MarshalJSON() ([]byte, error) {
	return []byte(`"` + gh.IBytes(uint64(b)) + `"`), nil
}

// UnmarshalJSON parses a JSON string into the bytesize.
func (b *ByteSize) UnmarshalJSON(text []byte) error {
	s, err := strconv.Unquote(string(text))
	if err != nil {
		return errors.WithStack(err)
	}
	v, err := gh.ParseBytes(s)
	if err != nil {
		return errors.WithStack(err)
	}
	*b = ByteSize(v)
	return nil
}

// UnmarshalText parses a Toml string into the bytesize.
func (b *ByteSize) UnmarshalText(text []byte) error {
	v, err := gh.ParseBytes(string(text))
	if err != nil {
		return errors.WithStack(err)
	}
	*b = ByteSize(v)
	return nil
}

//StringSlice is more friendly to json encode/decode
type StringSlice []string

// MarshalJSON returns the size as a JSON string.
func (s StringSlice) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strings.Join(s, ","))), nil
}

// UnmarshalJSON parses a JSON string into the bytesize.
func (s *StringSlice) UnmarshalJSON(text []byte) error {
	data, err := strconv.Unquote(string(text))
	if err != nil {
		return errors.WithStack(err)
	}
	if len(data) == 0 {
		*s = nil
		return nil
	}
	*s = strings.Split(data, ",")
	return nil
}

const (
	defaultLeaderLease             = int64(3)
	defaultNextRetryDelay          = time.Second
	defaultCompactionMode          = "periodic"
	defaultAutoCompactionRetention = "1h"

	defaultName                = "pd"
	defaultClientUrls          = "http://127.0.0.1:2379"
	defaultPeerUrls            = "http://127.0.0.1:2380"
	defaultInitialClusterState = "new"

	// etcd use 100ms for heartbeat and 1s for election timeout.
	// We can enlarge both a little to reduce the network aggression.
	// now embed etcd use TickMs for heartbeat, we will update
	// after embed etcd decouples tick and heartbeat.
	defaultTickInterval = 500 * time.Millisecond
	// embed etcd has a check that `5 * tick > election`
	defaultElectionInterval = 3000 * time.Millisecond

	defaultHeartbeatStreamRebindInterval = time.Minute

	defaultLeaderPriorityCheckInterval = time.Minute

	defaultDisableErrorVerbose = true
)

func adjustString(v *string, defValue string) {
	if len(*v) == 0 {
		*v = defValue
	}
}

func adjustUint64(v *uint64, defValue uint64) {
	if *v == 0 {
		*v = defValue
	}
}

func adjustInt64(v *int64, defValue int64) {
	if *v == 0 {
		*v = defValue
	}
}

func adjustFloat64(v *float64, defValue float64) {
	if *v == 0 {
		*v = defValue
	}
}

func adjustDuration(v *Duration, defValue time.Duration) {
	if v.Duration == 0 {
		v.Duration = defValue
	}
}

func adjustSchedulers(v *SchedulerConfigs, defValue SchedulerConfigs) {
	if len(*v) == 0 {
		*v = defValue
	}
}

// Utility to test if a configuration is defined.
type configMetaData struct {
	meta *toml.MetaData
	path []string
}

func newConfigMetadata(meta *toml.MetaData) *configMetaData {
	return &configMetaData{meta: meta}
}

func (m *configMetaData) IsDefined(key string) bool {
	if m.meta == nil {
		return false
	}
	keys := append([]string(nil), m.path...)
	keys = append(keys, key)
	return m.meta.IsDefined(keys...)
}

func (m *configMetaData) Child(path ...string) *configMetaData {
	newPath := append([]string(nil), m.path...)
	newPath = append(newPath, path...)
	return &configMetaData{
		meta: m.meta,
		path: newPath,
	}
}

func (m *configMetaData) CheckUndecoded() error {
	if m.meta == nil {
		return nil
	}
	undecoded := m.meta.Undecoded()
	if len(undecoded) == 0 {
		return nil
	}
	errInfo := "Config contains undefined item: "
	for _, key := range undecoded {
		errInfo += key.String() + ", "
	}
	return errors.New(errInfo[:len(errInfo)-2])
}

// Adjust is used to adjust the PD configurations.
func (c *PdConfig) Adjust(meta *toml.MetaData) error {
	configMetaData := newConfigMetadata(meta)
	if err := configMetaData.CheckUndecoded(); err != nil {
		c.WarningMsgs = append(c.WarningMsgs, err.Error())
	}
	adjustString(&c.Name, defaultName)
	adjustString(&c.DataDir, fmt.Sprintf("default.%s", c.Name))

	if err := c.validate(); err != nil {
		return err
	}

	adjustString(&c.ClientUrls, defaultClientUrls)
	adjustString(&c.AdvertiseClientUrls, c.ClientUrls)
	adjustString(&c.PeerUrls, defaultPeerUrls)
	adjustString(&c.AdvertisePeerUrls, c.PeerUrls)

	if len(c.InitialCluster) == 0 {
		// The advertise peer urls may be http://127.0.0.1:2380,http://127.0.0.1:2381
		// so the initial cluster is pd=http://127.0.0.1:2380,pd=http://127.0.0.1:2381
		items := strings.Split(c.AdvertisePeerUrls, ",")

		sep := ""
		for _, item := range items {
			c.InitialCluster += fmt.Sprintf("%s%s=%s", sep, c.Name, item)
			sep = ","
		}
	}

	adjustString(&c.InitialClusterState, defaultInitialClusterState)

	if len(c.Join) > 0 {
		if _, err := url.Parse(c.Join); err != nil {
			return errors.Errorf("failed to parse join addr:%s, err:%v", c.Join, err)
		}
	}

	adjustInt64(&c.LeaderLease, defaultLeaderLease)

	adjustDuration(&c.TsoSaveInterval, time.Duration(defaultLeaderLease)*time.Second)

	if c.nextRetryDelay == 0 {
		c.nextRetryDelay = defaultNextRetryDelay
	}

	adjustString(&c.AutoCompactionMode, defaultCompactionMode)
	adjustString(&c.AutoCompactionRetention, defaultAutoCompactionRetention)
	adjustDuration(&c.TickInterval, defaultTickInterval)
	adjustDuration(&c.ElectionInterval, defaultElectionInterval)

	adjustString(&c.NamespaceClassifier, "table")

	adjustString(&c.Metric.PushJob, c.Name)

	if err := c.Schedule.adjust(configMetaData.Child("schedule")); err != nil {
		return err
	}
	if err := c.Replication.adjust(configMetaData.Child("replication")); err != nil {
		return err
	}

	c.adjustLog(configMetaData.Child("log"))

	adjustDuration(&c.heartbeatStreamBindInterval, defaultHeartbeatStreamRebindInterval)

	adjustDuration(&c.leaderPriorityCheckInterval, defaultLeaderPriorityCheckInterval)

	if !configMetaData.IsDefined("enable-prevote") {
		c.PreVote = true
	}
	return nil
}

func (c *PdConfig) adjustLog(meta *configMetaData) {
	if !meta.IsDefined("disable-error-verbose") {
		c.Log.DisableErrorVerbose = defaultDisableErrorVerbose
	}
}

func (c *PdConfig) clone() *PdConfig {
	cfg := &PdConfig{}
	*cfg = *c
	return cfg
}

func (c *PdConfig) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "<nil>"
	}
	return string(data)
}

const (
	defaultMaxReplicas            = 3
	defaultMaxSnapshotCount       = 3
	defaultMaxPendingPeerCount    = 16
	defaultMaxMergeRegionSize     = 0
	defaultMaxMergeRegionKeys     = 0
	defaultSplitMergeInterval     = 1 * time.Hour
	defaultPatrolRegionInterval   = 100 * time.Millisecond
	defaultMaxStoreDownTime       = 30 * time.Minute
	defaultLeaderScheduleLimit    = 4
	defaultRegionScheduleLimit    = 4
	defaultReplicaScheduleLimit   = 8
	defaultMergeScheduleLimit     = 8
	defaultHotRegionScheduleLimit = 2
	defaultTolerantSizeRatio      = 5
	defaultLowSpaceRatio          = 0.8
	defaultHighSpaceRatio         = 0.6
	// defaultHotRegionCacheHitsThreshold is the low hit number threshold of the
	// hot region.
	defaultHotRegionCacheHitsThreshold = 3
)

func (c *ScheduleConfig) adjust(meta *configMetaData) error {
	if !meta.IsDefined("max-snapshot-count") {
		adjustUint64(&c.MaxSnapshotCount, defaultMaxSnapshotCount)
	}
	if !meta.IsDefined("max-pending-peer-count") {
		adjustUint64(&c.MaxPendingPeerCount, defaultMaxPendingPeerCount)
	}
	if !meta.IsDefined("max-merge-region-size") {
		adjustUint64(&c.MaxMergeRegionSize, defaultMaxMergeRegionSize)
	}
	if !meta.IsDefined("max-merge-region-keys") {
		adjustUint64(&c.MaxMergeRegionKeys, defaultMaxMergeRegionKeys)
	}
	adjustDuration(&c.SplitMergeInterval, defaultSplitMergeInterval)
	adjustDuration(&c.PatrolRegionInterval, defaultPatrolRegionInterval)
	adjustDuration(&c.MaxStoreDownTime, defaultMaxStoreDownTime)
	if !meta.IsDefined("leader-schedule-limit") {
		adjustUint64(&c.LeaderScheduleLimit, defaultLeaderScheduleLimit)
	}
	if !meta.IsDefined("region-schedule-limit") {
		adjustUint64(&c.RegionScheduleLimit, defaultRegionScheduleLimit)
	}
	if !meta.IsDefined("replica-schedule-limit") {
		adjustUint64(&c.ReplicaScheduleLimit, defaultReplicaScheduleLimit)
	}
	if !meta.IsDefined("merge-schedule-limit") {
		adjustUint64(&c.MergeScheduleLimit, defaultMergeScheduleLimit)
	}
	if !meta.IsDefined("hot-region-schedule-limit") {
		adjustUint64(&c.HotRegionScheduleLimit, defaultHotRegionScheduleLimit)
	}
	if !meta.IsDefined("hot-region-cache-hits-threshold") {
		adjustUint64(&c.HotRegionCacheHitsThreshold, defaultHotRegionCacheHitsThreshold)
	}
	if !meta.IsDefined("tolerant-size-ratio") {
		adjustFloat64(&c.TolerantSizeRatio, defaultTolerantSizeRatio)
	}
	adjustFloat64(&c.LowSpaceRatio, defaultLowSpaceRatio)
	adjustFloat64(&c.HighSpaceRatio, defaultHighSpaceRatio)
	adjustSchedulers(&c.Schedulers, defaultSchedulers)

	return c.validate()
}

func (c *ScheduleConfig) validate() error {
	if c.TolerantSizeRatio < 0 {
		return errors.New("tolerant-size-ratio should be nonnegative")
	}
	if c.LowSpaceRatio < 0 || c.LowSpaceRatio > 1 {
		return errors.New("low-space-ratio should between 0 and 1")
	}
	if c.HighSpaceRatio < 0 || c.HighSpaceRatio > 1 {
		return errors.New("high-space-ratio should between 0 and 1")
	}
	if c.LowSpaceRatio <= c.HighSpaceRatio {
		return errors.New("low-space-ratio should be larger than high-space-ratio")
	}
	return nil
}

func (c *PdConfig) validate() error {
	if c.Join != "" && c.InitialCluster != "" {
		return errors.New("-initial-cluster and -join can not be provided at the same time")
	}
	dataDir, err := filepath.Abs(c.DataDir)
	if err != nil {
		return errors.WithStack(err)
	}
	logFile, err := filepath.Abs(c.Log.File.Filename)
	if err != nil {
		return errors.WithStack(err)
	}
	rel, err := filepath.Rel(dataDir, filepath.Dir(logFile))
	if err != nil {
		return errors.WithStack(err)
	}
	if !strings.HasPrefix(rel, "..") {
		return errors.New("log directory shouldn't be the subdirectory of data directory")
	}

	return nil
}

func (c *ReplicationConfig) validate() error {
	for _, label := range c.LocationLabels {
		err := ValidateLabelString(label)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ReplicationConfig) adjust(meta *configMetaData) error {
	adjustUint64(&c.MaxReplicas, defaultMaxReplicas)
	return c.validate()
}

const matchRule = "^[A-Za-z0-9]([-A-Za-z0-9_.]*[A-Za-z0-9])?$"

// ValidateLabelString checks the legality of the label string.
// The valid label consist of alphanumeric characters, '-', '_' or '.',
// and must start and end with an alphanumeric character.
func ValidateLabelString(s string) error {
	isValid, _ := regexp.MatchString(matchRule, s)
	if !isValid {
		return errors.Errorf("invalid label: %s", s)
	}
	return nil
}

// ValidateLabels checks the legality of the labels.
func ValidateLabels(labels []*metapb.StoreLabel) error {
	for _, label := range labels {
		err := ValidateLabelString(label.Key)
		if err != nil {
			return err
		}
		err = ValidateLabelString(label.Value)
		if err != nil {
			return err
		}
	}
	return nil
}
