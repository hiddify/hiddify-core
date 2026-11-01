package hcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/experimental/libbox"
	E "github.com/sagernet/sing/common/exceptions"
)

// Diagnostics: what the app sends the developers after a crash. Everything comes from this
// core's working directory and process, so the app asks each core (its own, and the background
// one when it runs in another process) and merges what they return.

// reportDirectories are the archives libbox keeps in the working directory: a directory per
// report, named by its time, with go.log (the crash) or the OOM/power files and metadata.json.
var reportDirectories = []string{"crash_reports", "oom_reports", "power_reports"}

// liveCrashFiles are where the cores' stderr goes while they run (data/stderr<mode>.log from
// Setup, stderr.log and stderr2.log from Android, CrashReport-.log from libbox, and the iOS
// extension's error file). A crash in them is archived at the next start of the same core; until
// then, only the file tells.
var liveCrashFiles = []string{
	"data/stderr0.log", "data/stderr1.log", "data/stderr2.log", "data/stderr3.log", "data/stderr4.log",
	"stderr.log", "stderr2.log", "CrashReport-.log", "network_extension_error.log",
}

// crashMarker is what a Go crash writes to stderr: a panic, a fatal error or a signal. Other
// stderr output (a library's warning) is not a crash.
var crashMarker = regexp.MustCompile(`(?m)^(panic: |fatal error: |runtime: |SIG[A-Z]+: |\[signal )|goroutine \d+ \[`)

// The goroutines that matter after a crash are the crashed run's, and a run killed by the
// system (an OOM kill, iOS ending the extension) writes nothing. So while the core is connected,
// its goroutines are written to goroutineSnapshotPath every goroutineSnapshotInterval, and the
// next Setup keeps the previous run's file aside as goroutinePreviousPath for the zip. A Go
// panic also has every goroutine in its crash report (debug.SetTraceback("all")).
const goroutineSnapshotInterval = time.Minute

func goroutineSnapshotPath(workingPath string, mode SetupMode) string {
	return filepath.Join(workingPath, "data", fmt.Sprintf("goroutines-%s.txt", strings.ToLower(mode.String())))
}

func goroutinePreviousPath(workingPath string, mode SetupMode) string {
	return filepath.Join(workingPath, "data", fmt.Sprintf("goroutines-%s.previous.txt", strings.ToLower(mode.String())))
}

// keepPreviousGoroutineSnapshot moves the last run's snapshot aside at Setup, before this run
// writes its own; a run that wrote none leaves the previous one in place.
func keepPreviousGoroutineSnapshot(workingPath string, mode SetupMode) {
	snapshot := goroutineSnapshotPath(workingPath, mode)
	if _, err := os.Stat(snapshot); err != nil {
		return
	}
	os.Rename(snapshot, goroutinePreviousPath(workingPath, mode))
}

// writeGoroutineSnapshots writes the snapshot every interval while the core is running, until
// ctx ends. Setup starts it once per process.
func writeGoroutineSnapshots(ctx context.Context, workingPath string, mode SetupMode) {
	ticker := time.NewTicker(goroutineSnapshotInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if !slices.Contains(running, static.CoreState) {
			continue
		}
		os.MkdirAll(filepath.Join(workingPath, "data"), 0o755)
		os.WriteFile(goroutineSnapshotPath(workingPath, mode), []byte(time.Now().UTC().Format(time.RFC3339)+"\n\n"+libbox.GoroutineDump()), 0o644)
	}
}

// maxDiagnosticsReports is how many reports of each kind the zip holds, newest first.
const maxDiagnosticsReports = 5

// maxDiagnosticsLogSize is the longest tail of a log file the zip holds.
const maxDiagnosticsLogSize = 2 << 20

func (s *CoreService) ListCrashReports(ctx context.Context, req *hcommon.Empty) (*CrashReportList, error) {
	return &CrashReportList{Reports: listCrashReports(sWorkingPath)}, nil
}

func listCrashReports(workingPath string) []*CrashReport {
	var reports []*CrashReport
	for _, directory := range reportDirectories[:2] { // power reports are not crashes
		entries, err := os.ReadDir(filepath.Join(workingPath, directory))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			report := &CrashReport{Name: entry.Name(), Source: directory, CrashedAt: reportTime(entry.Name())}
			content, _ := os.ReadFile(filepath.Join(workingPath, directory, entry.Name(), "go.log"))
			report.IsCrash = directory == "oom_reports" || crashMarker.Match(content)
			reports = append(reports, report)
		}
	}
	for _, name := range liveCrashFiles {
		path := filepath.Join(workingPath, name)
		content, err := os.ReadFile(path)
		if err != nil || len(strings.TrimSpace(string(content))) == 0 {
			continue
		}
		info, _ := os.Stat(path)
		report := &CrashReport{Name: name, Source: name, IsCrash: crashMarker.Match(content)}
		if info != nil {
			report.CrashedAt = info.ModTime().UTC().Format(time.RFC3339)
		}
		reports = append(reports, report)
	}
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].CrashedAt < reports[j].CrashedAt })
	return reports
}

// reportTime reads the time out of a report directory's name, 2006-01-02T15-04-05[-n].
func reportTime(name string) string {
	if len(name) < 19 {
		return ""
	}
	parsed, err := time.Parse("2006-01-02T15-04-05", name[:19])
	if err != nil {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339)
}

func (s *CoreService) ExportDiagnostics(ctx context.Context, req *ExportDiagnosticsRequest) (*ExportDiagnosticsResponse, error) {
	path, err := exportDiagnostics(sWorkingPath, sTempPath, req)
	if err != nil {
		return nil, err
	}
	response := &ExportDiagnosticsResponse{Path: path}
	if req.Inline {
		response.Content, err = os.ReadFile(path)
		if err != nil {
			return nil, E.Cause(err, "read the diagnostics zip")
		}
	}
	return response, nil
}

// exportDiagnostics gathers the files into a directory under tempPath and zips it with
// libbox.CreateZipArchive, which scrubs text files of addresses, links and secrets.
func exportDiagnostics(workingPath, tempPath string, req *ExportDiagnosticsRequest) (string, error) {
	stamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	staging := filepath.Join(tempPath, "diagnostics-"+stamp)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", E.Cause(err, "create the diagnostics directory")
	}
	defer os.RemoveAll(staging)

	for _, directory := range reportDirectories {
		copyNewestReports(filepath.Join(workingPath, directory), filepath.Join(staging, directory))
	}
	for _, name := range liveCrashFiles {
		copyLogTail(filepath.Join(workingPath, name), filepath.Join(staging, "logs", strings.ReplaceAll(name, "/", "-")))
	}
	copyLogTail(filepath.Join(workingPath, "data", "box.log"), filepath.Join(staging, "logs", "box.log"))
	if content, err := os.ReadFile(filepath.Join(workingPath, "data", "current-config.json")); err == nil {
		if sanitized := config.SanitizeConfigJSON(content); sanitized != nil {
			os.WriteFile(filepath.Join(staging, "config.json"), sanitized, 0o644)
		}
	}
	for _, mode := range []SetupMode{SetupMode_OLD, SetupMode_GRPC_NORMAL, SetupMode_GRPC_BACKGROUND, SetupMode_GRPC_NORMAL_INSECURE, SetupMode_GRPC_BACKGROUND_INSECURE} {
		// the previous run's goroutines of every core that shares this working directory
		copyLogTail(goroutinePreviousPath(workingPath, mode), filepath.Join(staging, "goroutines", filepath.Base(goroutinePreviousPath(workingPath, mode))))
	}
	if len(req.ExtraFiles) > 0 {
		os.MkdirAll(filepath.Join(staging, "extra"), 0o755)
		for name, content := range req.ExtraFiles {
			os.WriteFile(filepath.Join(staging, "extra", filepath.Base(name)), content, 0o644)
		}
	}
	metadata, _ := json.MarshalIndent(map[string]any{
		"exportedAt":  time.Now().UTC().Format(time.RFC3339),
		"mode":        sSetupMode.String(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"goVersion":   runtime.Version(),
		"coreState":   static.CoreState.String(),
		"reports":     listCrashReports(workingPath),
		"workingPath": filepath.Base(workingPath),
	}, "", "  ")
	os.WriteFile(filepath.Join(staging, "metadata.json"), metadata, 0o644)

	destination := req.DestinationPath
	if destination == "" {
		destination = filepath.Join(tempPath, "hiddify-diagnostics-"+stamp+".zip")
	}
	if err := libbox.CreateZipArchive(staging, destination, false); err != nil {
		return "", E.Cause(err, "zip the diagnostics")
	}
	return destination, nil
}

// copyNewestReports copies the newest maxDiagnosticsReports report directories of source.
func copyNewestReports(source, destination string) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names) // named by time
	if len(names) > maxDiagnosticsReports {
		names = names[len(names)-maxDiagnosticsReports:]
	}
	for _, name := range names {
		copyDirectory(filepath.Join(source, name), filepath.Join(destination, name))
	}
}

func copyDirectory(source, destination string) {
	filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		relative, _ := filepath.Rel(source, path)
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			os.MkdirAll(target, 0o755)
			return nil
		}
		copyLogTail(path, target)
		return nil
	})
}

// copyLogTail copies the last maxDiagnosticsLogSize bytes of source, when it has any.
func copyLogTail(source, destination string) {
	file, err := os.Open(source)
	if err != nil {
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() == 0 {
		return
	}
	if info.Size() > maxDiagnosticsLogSize {
		file.Seek(info.Size()-maxDiagnosticsLogSize, io.SeekStart)
	}
	os.MkdirAll(filepath.Dir(destination), 0o755)
	target, err := os.Create(destination)
	if err != nil {
		return
	}
	defer target.Close()
	if info.Size() > maxDiagnosticsLogSize {
		fmt.Fprintf(target, "... the last %d bytes of %d ...\n", maxDiagnosticsLogSize, info.Size())
	}
	io.Copy(target, file)
}
