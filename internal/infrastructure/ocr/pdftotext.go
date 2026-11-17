package ocr

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
)

// popplerMemoryLimitKB caps the address space of poppler subprocesses. A
// malformed PDF can make a parser allocate without bound; outside the API
// process that only kills the helper, not the server.
const popplerMemoryLimitKB = 2 * 1024 * 1024

const maxPDFTextOutput = 32 * 1024 * 1024

// PopplerTextExtractor reads a PDF's embedded text layer with `pdftotext`
// (poppler-utils) in a separate, memory-capped process that is killed when
// the context ends.
type PopplerTextExtractor struct {
	binaryPath string
	available  bool
}

func NewPopplerTextExtractor(binaryPath string) *PopplerTextExtractor {
	if binaryPath == "" {
		binaryPath = "pdftotext"
	}

	_, err := exec.LookPath(binaryPath)

	return &PopplerTextExtractor{
		binaryPath: binaryPath,
		available:  err == nil,
	}
}

func (p *PopplerTextExtractor) Available() bool { return p != nil && p.available }

func (p *PopplerTextExtractor) Extract(ctx context.Context, pdfPath string) (string, int, error) {
	if !p.Available() {
		return "", 0, fault.Wrap(ErrEngineUnavailable, fctx.With(ctx), fmsg.With("pdftotext binary not found"))
	}

	var stdout cappedBuffer
	stdout.limit = maxPDFTextOutput
	var stderr bytes.Buffer

	cmd := memoryLimitedCommand(ctx, popplerMemoryLimitKB, p.binaryPath, "-enc", "UTF-8", "-eol", "unix", pdfPath, "-")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, fault.Wrap(err, fctx.With(ctx), fmsg.With("pdftotext failed: "+strings.TrimSpace(stderr.String())))
	}

	out := stdout.String()
	pages := strings.Count(out, "\f")

	return strings.ReplaceAll(out, "\f", "\n"), pages, nil
}

// memoryLimitedCommand runs bin under `ulimit -v` where a POSIX shell is
// available. If the shell cannot apply the limit the command still runs.
func memoryLimitedCommand(ctx context.Context, limitKB int, bin string, args ...string) *exec.Cmd {
	if runtime.GOOS != "linux" {
		return exec.CommandContext(ctx, bin, args...)
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		return exec.CommandContext(ctx, bin, args...)
	}

	script := "ulimit -v " + strconv.Itoa(limitKB) + " 2>/dev/null; exec \"$@\""
	shArgs := append([]string{"-c", script, "sh", bin}, args...)

	return exec.CommandContext(ctx, sh, shArgs...)
}

// cappedBuffer keeps the first limit bytes and silently drops the rest, so a
// runaway subprocess can't grow the API process's memory through its output.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room > 0 {
		if len(p) > room {
			b.Buffer.Write(p[:room])
		} else {
			b.Buffer.Write(p)
		}
	}
	return len(p), nil
}
