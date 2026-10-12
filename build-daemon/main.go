// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ImBIOS/sidekick/internal/fleet"
)

// Runner executes one claimed build. execRunner shells xcodebuild;
// tests substitute fakeRunner. Keeping execution behind an interface is
// what lets CI verify the whole claim→logs→finish loop without a Mac.
type Runner interface {
	// Run executes scheme/ref in workdir, calling emit for each output
	// line as it arrives. It returns the process exit error, if any.
	Run(workdir, scheme, ref string, emit func(string)) error
}

// poster is the slice of the fleet client driveBuild needs. The real
// *fleet.Client satisfies it; tests use memPoster.
type poster interface {
	PostBuildLogs(id string, lines []string) error
	FinishBuild(id, status string) error
}

// execRunner runs real xcodebuild with a warm DerivedData path. If workdir
// is a git checkout and ref names a revision, it checks the revision out
// first so remote builds track what the agent pushed.
type execRunner struct {
	derivedData string
}

func (r execRunner) Run(workdir, scheme, ref string, emit func(string)) error {
	if ref != "" {
		if _, err := os.Stat(workdir + "/.git"); err == nil {
			out, err := exec.Command("git", "-C", workdir, "fetch", "--quiet").CombinedOutput()
			emit("$ git fetch")
			emitStrings(emit, out)
			if err != nil {
				return fmt.Errorf("git fetch: %w", err)
			}
			out, err = exec.Command("git", "-C", workdir, "checkout", "--quiet", ref).CombinedOutput()
			emit("$ git checkout " + ref)
			emitStrings(emit, out)
			if err != nil {
				return fmt.Errorf("git checkout %s: %w", ref, err)
			}
		}
	}
	args := []string{"-scheme", scheme, "-destination", "generic/platform=iOS Simulator"}
	if r.derivedData != "" {
		args = append(args, "-derivedDataPath", r.derivedData)
	}
	args = append(args, "build")
	cmd := exec.Command("xcodebuild", args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	emit("$ xcodebuild " + strings.Join(args, " "))
	emitStrings(emit, out)
	return err
}

func emitStrings(emit func(string), out []byte) {
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			emit(line)
		}
	}
}

// driveBuild runs one claimed build to terminal state, streaming log
// batches as they arrive so agents watch progress live.
func driveBuild(fc poster, build map[string]any, r Runner, workdir string) {
	id, _ := build["build_id"].(string)
	scheme, _ := build["scheme"].(string)
	ref, _ := build["ref"].(string)
	var mu sync.Mutex
	var batch []string
	flush := func() {
		mu.Lock()
		defer mu.Unlock()
		if len(batch) == 0 {
			return
		}
		_ = fc.PostBuildLogs(id, batch)
		batch = nil
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	done := make(chan error, 1)
	go func() {
		done <- r.Run(workdir, scheme, ref, func(line string) {
			mu.Lock()
			batch = append(batch, line)
			mu.Unlock()
		})
	}()
	for {
		select {
		case err := <-done:
			flush()
			if err != nil {
				_ = fc.PostBuildLogs(id, []string{"[sidekick] build failed: " + err.Error()})
				_ = fc.FinishBuild(id, "failed")
			} else {
				_ = fc.PostBuildLogs(id, []string{"[sidekick] build succeeded"})
				_ = fc.FinishBuild(id, "succeeded")
			}
			return
		case <-tick.C:
			flush()
		}
	}
}

func main() {
	control := flag.String("control", "", "control-plane URL (or SIDEKICK_CONTROL)")
	token := flag.String("token", "", "fleet host token (or SIDEKICK_HOST_TOKEN)")
	pool := flag.String("pool", "", "pool name (or SIDEKICK_POOL)")
	workdir := flag.String("workdir", ".", "project checkout xcodebuild runs in")
	derived := flag.String("derived", "", "warm DerivedData path (empty = xcodebuild default)")
	interval := flag.Duration("interval", 2*time.Second, "claim poll interval")
	flag.Parse()

	ctrl, tok, pl, err := fleet.Resolve(*control, *token, *pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "build-daemon:", err)
		os.Exit(1)
	}
	fc := fleet.New(ctrl, tok)
	fmt.Printf("build-daemon: pool %s via %s (workdir %s)\n", pl, ctrl, *workdir)
	r := execRunner{derivedData: *derived}
	for {
		claim, ok, err := fc.ClaimNext(pl, []string{"build"})
		if err != nil {
			fmt.Fprintln(os.Stderr, "build-daemon: claim:", err)
			time.Sleep(*interval)
			continue
		}
		if !ok {
			time.Sleep(*interval)
			continue
		}
		driveBuild(fc, claim.Build, r, *workdir)
	}
}
