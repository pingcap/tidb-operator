package compact

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/klog/v2"
)

// CompactCtx is the context for *a* compaction.
type CompactCtx struct {
	kopts options.KubeOpts
	opts  options.CompactOpts

	// Delegate defines some hooks that allow you to listen over events and
	// controls what compactions should be done.
	Delegate Delegate

	/* Below fields are used for testing. */

	// OverrideBase64ify overrides the shell command used for generate
	// the storage base64.
	OverrideBase64ify func(ctx context.Context, args []string) *exec.Cmd
	// OverrideCompact overrides the shell command used for compacting.
	OverrideCompact func(ctx context.Context, args []string) *exec.Cmd
}

type Progress struct {
	MetaCompleted  uint64 `json:"meta_completed"`
	MetaTotal      uint64 `json:"meta_total"`
	BytesToCompact uint64 `json:"bytes_to_compact"`
	BytesCompacted uint64 `json:"bytes_compacted"`
}

type CompactionRef struct {
	Namespace string
	Name      string
}

// Delegate describes the "environment" that a compaction runs.
type Delegate interface {
	// GetCompaction will be called at the very beginning.
	// This should return a set of parameter of a compaction, then a compaction
	// defined by these parameters will be executed.
	//
	// The argument `ref` refs to a `CompactBackup` resource in the cluster,
	// and the compaction should be generated according to it.
	GetCompaction(ctx context.Context, ref CompactionRef) (options.CompactOpts, error)

	// OnStart will be called when the `tikv-ctl compact-log-backup` process is about to be spawned.
	OnStart(ctx context.Context)
	// OnPrgress will be called when the progress of compaction updated.
	OnProgress(ctx context.Context, p Progress)
	// OnFinish will be called when the `tikv-ctl` process exits.
	OnFinish(ctx context.Context, err error)
}

// BaseDelegate is an empty delegate.
//
// You shouldn't directly use this delegate, instead you may embed it into
// your delegate implementation and override the methods you need.
type BaseDelegate struct{}

func (BaseDelegate) GetCompaction(ctx context.Context, ref CompactionRef) (options.CompactOpts, error) {
	return options.CompactOpts{}, errors.New("override `GetCompaction` to provide a meanful execution")
}
func (BaseDelegate) OnStart(ctx context.Context)                {}
func (BaseDelegate) OnProgress(ctx context.Context, p Progress) {}
func (BaseDelegate) OnFinish(ctx context.Context, err error)    {}

func (r *CompactCtx) brBin() string {
	return filepath.Join(util.BRBinPath, "br")
}

func (r *CompactCtx) kvCtlBin() string {
	return filepath.Join(util.KVCTLBinPath, "tikv-ctl")
}

func (r *CompactCtx) base64ifyCmd(ctx context.Context, extraArgs []string) *exec.Cmd {
	br := r.brBin()
	args := []string{
		"operator",
		"base64ify",
	}
	args = append(args, extraArgs...)

	if r.OverrideBase64ify != nil {
		return r.OverrideBase64ify(ctx, args)
	}
	return exec.CommandContext(ctx, br, args...)
}

func (r *CompactCtx) base64ifyStorage(ctx context.Context) (string, error) {
	brCmd := r.base64ifyCmd(ctx, r.opts.StorageOpts)
	out, err := brCmd.Output()
	if err != nil {
		eerr := err.(*exec.ExitError)
		klog.Warningf("Failed to execute base64ify; stderr = %s", string(eerr.Stderr))
		return "", errors.Annotatef(err, "failed to execute BR with args %v", brCmd.Args)
	}
	out = bytes.Trim(out, "\r\n \t")
	return string(out), nil
}

func (r *CompactCtx) compactCmd(ctx context.Context, base64Storage string) *exec.Cmd {
	ctl := r.kvCtlBin()
	args := []string{
		"--log-level",
		"INFO",
		"--log-format",
		"json",
		"compact-log-backup",
		"--storage-base64",
		base64Storage,
		"--from",
		strconv.FormatUint(r.opts.FromTS, 10),
		"--until",
		strconv.FormatUint(r.opts.UntilTS, 10),
		"-N",
		strconv.FormatUint(r.opts.Concurrency, 10),
	}

	if r.OverrideCompact != nil {
		return r.OverrideCompact(ctx, args)
	}
	return exec.CommandContext(ctx, ctl, args...)
}

func (r *CompactCtx) runCompaction(ctx context.Context, base64Storage string) (err error) {
	cmd := r.compactCmd(ctx, base64Storage)
	defer func() { r.Delegate.OnFinish(ctx, err) }()

	logs, err := cmd.StderrPipe()
	if err != nil {
		return errors.Annotate(err, "failed to create stderr pipe for compact")
	}
	if err := cmd.Start(); err != nil {
		return errors.Annotate(err, "failed to start compact")
	}

	r.Delegate.OnStart(ctx)
	err = r.processCompactionLogs(ctx, io.TeeReader(logs, os.Stdout))
	if err != nil {
		return err
	}

	return cmd.Wait()
}

// logLine is line of JSON log.
// It just extracted the message from the JSON and keeps the origin json bytes.
// So you may extract fields from it by `json.Unmarshal(l.Raw, ...)`.
type logLine struct {
	Message string

	// Raw is the original log JSON.
	Raw []byte
}

var _ json.Unmarshaler = &logLine{}

func (l *logLine) UnmarshalJSON(bytes []byte) error {
	item := struct {
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(bytes, &item); err != nil {
		return err
	}
	l.Message = item.Message
	l.Raw = bytes

	return nil
}

func (r *CompactCtx) processLogLine(ctx context.Context, l logLine) error {
	const (
		messageCompactionDone = "Finishing compaction."
		messageCompactAborted = "Compaction aborted."
	)

	switch l.Message {
	case messageCompactionDone:
		var prog Progress
		if err := json.Unmarshal(l.Raw, &prog); err != nil {
			return errors.Annotate(err, "failed to decode progress")
		}
		r.Delegate.OnProgress(ctx, prog)
		return nil
	case messageCompactAborted:
		errContainer := struct {
			Err string `json:"err"`
		}{}
		if err := json.Unmarshal(l.Raw, &errContainer); err != nil {
			return errors.Annotate(err, "failed to decode error message")
		}
		return errors.Errorf("compaction aborted: %s", errContainer.Err)
	default:
		return nil
	}
}

func (r *CompactCtx) processCompactionLogs(ctx context.Context, logStream io.Reader) error {
	dec := json.NewDecoder(logStream)

	var line logLine
	for dec.More() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := dec.Decode(&line); err != nil {
			return errors.Annotate(err, "failed to decode the line of log")
		}
		if err := r.processLogLine(ctx, line); err != nil {
			return errors.Annotate(err, "error during processing log line")
		}
	}

	return nil
}

func (r *CompactCtx) Run(ctx context.Context) error {
	ref := CompactionRef{
		Namespace: r.kopts.Namespace,
		Name:      r.kopts.ResourceName,
	}
	opts, err := r.Delegate.GetCompaction(ctx, ref)
	if err != nil {
		return errors.Annotate(err, "failed to get storage string")
	}
	r.opts = opts

	b64, err := r.base64ifyStorage(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to base64ify storage")
	}
	return r.runCompaction(ctx, b64)
}

// New creates a new compaction context.
func New(opts options.KubeOpts, dele Delegate) *CompactCtx {
	return &CompactCtx{
		kopts:    opts,
		Delegate: dele,
	}
}
