package compact_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"testing"
	"testing/iotest"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/stretchr/testify/require"
)

type callRecorder []call

type call struct {
	cmd   string
	attrs map[string]any
}

func (c *callRecorder) GetCompaction(ctx context.Context, ref compact.CompactionRef) (options.CompactOpts, error) {
	*c = append(*c, call{cmd: "GetCompaction", attrs: map[string]any{
		"ref": ref,
	}})
	return options.CompactOpts{}, nil
}

func (c *callRecorder) OnStart(ctx context.Context) {
	*c = append(*c, call{cmd: "OnStart"})
}

func (c *callRecorder) OnProgress(ctx context.Context, p compact.Progress) {
	*c = append(*c, call{cmd: "OnProgress", attrs: map[string]any{
		"MetaCompleted":  p.MetaCompleted,
		"MetaTotal":      p.MetaTotal,
		"BytesToCompact": p.BytesToCompact,
		"BytesCompacted": p.BytesCompacted,
	}})
}

func (c *callRecorder) OnFinish(ctx context.Context, err error) {
	*c = append(*c, call{cmd: "OnFinish", attrs: map[string]any{
		"error": err,
	}})
}

type dummyCommands struct {
	theBase64 string
	t         *testing.T
}

func (d dummyCommands) OverrideBase64ify(context.Context, []string) *exec.Cmd {
	cmd := exec.Command("cat", "-")
	cmd.Stdin = bytes.NewBufferString(d.theBase64)
	return cmd
}

func (d dummyCommands) OverrideCompact(_ context.Context, args []string) *exec.Cmd {
	storage := ""
	for i, arg := range args {
		if arg == "--storage-base64" && i+1 < len(args) {
			storage = args[i+1]
			break
		}
	}

	require.Equal(d.t, storage, d.theBase64, "%#v", args)
	return exec.Command("true")
}

type empty struct {
	compact.BaseDelegate
}

func (e empty) GetCompaction(context.Context, compact.CompactionRef) (options.CompactOpts, error) {
	return options.CompactOpts{}, nil
}

func TestNormal(t *testing.T) {
	cx := compact.New(options.KubeOpts{}, empty{})
	dc := dummyCommands{
		theBase64: "was yae",
		t:         t,
	}
	cx.OverrideBase64ify = dc.OverrideBase64ify
	cx.OverrideCompact = dc.OverrideCompact

	require.NoError(t, cx.Run(context.Background()))
}

type dummyProgress struct {
	progs []compact.Progress
	t     *testing.T
}

func (d dummyProgress) OverrideBase64ify(context.Context, []string) *exec.Cmd {
	return exec.Command("true")
}

func toMap(t *testing.T, v any) (res map[string]any) {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &res))
	return
}

func (d dummyProgress) OverrideCompact(_ context.Context, args []string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", "cat - >&2")
	rx, tx := io.Pipe()

	enc := json.NewEncoder(tx)
	go func() {
		for _, prog := range d.progs {
			m := toMap(d.t, prog)
			m["message"] = "Finishing compaction."
			require.NoError(d.t, enc.Encode(m))
		}
		tx.Close()
	}()

	// Simulating streaming progress.
	cmd.Stdin = iotest.OneByteReader(rx)
	return cmd
}

func TestWithProgress(t *testing.T) {
	rec := callRecorder{}
	cx := compact.New(options.KubeOpts{Namespace: "foo", ResourceName: "bar"}, &rec)
	dp := dummyProgress{
		progs: []compact.Progress{
			{MetaCompleted: 1, MetaTotal: 2, BytesToCompact: 3, BytesCompacted: 4},
			{MetaCompleted: 2, MetaTotal: 2, BytesToCompact: 7, BytesCompacted: 8},
			{MetaCompleted: 2, MetaTotal: 2, BytesToCompact: 8, BytesCompacted: 8},
		},
		t: t,
	}
	cx.OverrideBase64ify = dp.OverrideBase64ify
	cx.OverrideCompact = dp.OverrideCompact

	require.NoError(t, cx.Run(context.Background()))

	require.Equal(t, []call(rec), []call{
		{cmd: "GetCompaction", attrs: map[string]any{
			"ref": compact.CompactionRef{
				Namespace: "foo",
				Name:      "bar",
			},
		}},
		{cmd: "OnStart"},
		{cmd: "OnProgress", attrs: map[string]any{
			"MetaCompleted":  uint64(1),
			"MetaTotal":      uint64(2),
			"BytesToCompact": uint64(3),
			"BytesCompacted": uint64(4),
		}},
		{cmd: "OnProgress", attrs: map[string]any{
			"MetaCompleted":  uint64(2),
			"MetaTotal":      uint64(2),
			"BytesToCompact": uint64(7),
			"BytesCompacted": uint64(8),
		}},
		{cmd: "OnProgress", attrs: map[string]any{
			"MetaCompleted":  uint64(2),
			"MetaTotal":      uint64(2),
			"BytesToCompact": uint64(8),
			"BytesCompacted": uint64(8),
		}},
		{cmd: "OnFinish", attrs: map[string]any{
			"error": error(nil),
		}},
	})
}
