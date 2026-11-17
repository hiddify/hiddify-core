package hcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/hiddify/hiddify-core/v2/hcommon/constants"
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
var liveCrashFiles = append(setupStderrFiles(),
	"stderr.log", "stderr2.log", "CrashReport-.log", "network_extension_error.log",
)

// setupStderrFiles are the files Setup sends stderr to, one per mode and named for it as
// grpc_server.go writes them: data/stderrGRPC_NORMAL.log, not data/stderr1.log. The name map, not
// SetupMode.String: the enum descriptor is not ready while package variables are set.
func setupStderrFiles() []string {
	modes := make([]int32, 0, len(SetupMode_name))
	for mode := range SetupMode_name {
		modes = append(modes, mode)
	}
	slices.Sort(modes)
	files := make([]string, 0, len(modes))
	for _, mode := range modes {
		files = append(files, "data/stderr"+SetupMode_name[mode]+".log")
	}
	return files
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
		os.WriteFile(goroutineSnapshotPath(workingPath, mode), []byte(time.Now().UTC().Format(time.RFC3339)+"\n\n"+goroutineDump()), 0o644)
	}
}

// goroutineDump is the goroutine profile as text: each distinct stack once, with the number of
// goroutines on it.
//
// It is not runtime.Stack(all) (libbox.GoroutineDump), whose output also has each goroutine's
// state and wait time: that one prints with the runtime's crash traceback, and a goroutine it
// cannot unwind (one between Go and C, as gomobile's are) makes it hexdump the stack around the
// frame. With the frame outside the goroutine's stack the hexdump's length is negative and the
// runtime throws (go1.27.3, "unsafe.Slice: len out of range" in runtime.hexdumpWords), so the
// snapshot, taken every minute, ended the core after some hours. The profile unwinds silently
// and stops at such a frame.
func goroutineDump() string {
	var dump strings.Builder
	pprof.Lookup("goroutine").WriteTo(&dump, 1)
	return dump.String()
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

// appSummarySections are the sections of the app's app-info.json that metadata.json repeats, so
// the build, the system, the state and the settings read first: the settings come without their
// secrets, which the app leaves out (app_diagnostics.dart). The long ones (proxies, network) stay
// in sensitive/app/app-info.json only.
var appSummarySections = []string{"app", "system", "connection", "profiles", "core", "settings"}

// addAppSummary copies appSummarySections of the app's app-info.json into metadata, when the app
// sent one and it is JSON.
func addAppSummary(metadata map[string]any, appInfo []byte) {
	if len(appInfo) == 0 {
		return
	}
	var info map[string]json.RawMessage
	if json.Unmarshal(appInfo, &info) != nil {
		return
	}
	for _, section := range appSummarySections {
		if value, ok := info[section]; ok {
			if _, taken := metadata[section]; !taken {
				metadata[section] = value
			}
		}
	}
}

// readableJSON is value as indented JSON in plain UTF-8: <set> stays <set>, not \u003cset\u003e,
// which encoding/json writes by default for HTML.
func readableJSON(value any) []byte {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(value)
	return buffer.Bytes()
}

// appRelativePath is path from its first folder named for Hiddify on: the app's own folders
// without the user's name and home above them, apple.hiddify.com or app.hiddify.com/files. A path
// with no such folder gives its last element.
func appRelativePath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(clean, "/")
	for i, part := range parts {
		if strings.Contains(strings.ToLower(part), "hiddify") {
			return strings.Join(parts[i:], "/")
		}
	}
	return filepath.Base(path)
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

// writeGenericFiles copies what is read without a key into directory: the goroutine snapshots,
// the core's crash reports and the caller's generic files.
func writeGenericFiles(workingPath, directory string, appFiles map[string][]byte) {
	copyNewestReports(filepath.Join(workingPath, "crash_reports"), filepath.Join(directory, "crash_reports"))
	for _, mode := range []SetupMode{SetupMode_OLD, SetupMode_GRPC_NORMAL, SetupMode_GRPC_BACKGROUND, SetupMode_GRPC_NORMAL_INSECURE, SetupMode_GRPC_BACKGROUND_INSECURE} {
		// the previous run's goroutines of every core that shares this working directory
		copyLogTail(goroutinePreviousPath(workingPath, mode), filepath.Join(directory, "goroutines", filepath.Base(goroutinePreviousPath(workingPath, mode))))
	}
	writeAppFiles(filepath.Join(directory, "app"), appFiles)
}

// writeSensitiveFiles copies what goes in the sensitive part of the zip into directory: the
// OOM and power reports, the logs, the config and the caller's other files.
func writeSensitiveFiles(workingPath, directory string, appFiles map[string][]byte) {
	for _, reports := range []string{"oom_reports", "power_reports"} {
		copyNewestReports(filepath.Join(workingPath, reports), filepath.Join(directory, reports))
	}
	for _, name := range liveCrashFiles {
		copyLogTail(filepath.Join(workingPath, name), filepath.Join(directory, "logs", strings.ReplaceAll(name, "/", "-")))
	}
	copyLogTail(filepath.Join(workingPath, "data", "box.log"), filepath.Join(directory, "logs", "box.log"))
	os.MkdirAll(directory, 0o755)
	os.WriteFile(filepath.Join(directory, "network.json"), readableJSON(networkDiagnostics()), 0o644)
	if content, err := os.ReadFile(filepath.Join(workingPath, "data", "current-config.json")); err == nil {
		if sanitized := config.SanitizeConfigJSON(content); sanitized != nil {
			os.MkdirAll(directory, 0o755)
			os.WriteFile(filepath.Join(directory, "config.json"), sanitized, 0o644)
		}
	}
	writeAppFiles(filepath.Join(directory, "app"), appFiles)
}

// writeAppFiles writes the caller's files into directory, by their base name only.
func writeAppFiles(directory string, files map[string][]byte) {
	if len(files) == 0 {
		return
	}
	os.MkdirAll(directory, 0o755)
	for name, content := range files {
		os.WriteFile(filepath.Join(directory, filepath.Base(name)), content, 0o644)
	}
}

// A diagnostics report has two parts. The generic one is read by anyone who gets the report;
// the sensitive one is encrypted to diagnosticsPublicKey (diagnostics_key.go), so only the Hiddify
// team opens it:
//
//	diagnostics-<time>/
//	  README.txt               what is encrypted and what is not, with the encrypted files' names
//	  metadata.json            generic: the build, the system, the core's state, the list of crash
//	                           reports and, from the app (app-info.json), its version, the
//	                           connection state and the settings without secrets
//	  goroutines/              generic: the previous run's goroutine snapshots, stacks of code
//	  crash_reports/           generic: the core's panics and fatal errors with their stacks
//	  app/                     generic: what the caller adds as GenericFiles, the app's own
//	                           exceptions with their stack traces
//	  sensitive.encrypted.age  an age-encrypted zip of what may say who the user is or where they
//	                           connect:
//	    config.json            the running config, servers and keys removed
//	    network.json           the interface the core binds its connections to and, on Windows,
//	                           every interface's routing state (diagnostics_network.go)
//	    logs/                  box.log and the cores' stderr
//	    oom_reports/ power_reports/
//	    app/                   what the caller adds (ExtraFiles): the app's logs and crashes, the
//	                           user's description, the full app-info.json, another core's report
//
// Asked unencrypted, or in a build without a key, sensitive/ is a plain folder instead of
// sensitive.encrypted.age. A report made by the core alone has what the core knows; the app adds
// its own files. Text in both parts is scrubbed of addresses, links and secrets as it is zipped.
const (
	sensitiveDirectory = "sensitive"
	encryptedSensitive = "sensitive.encrypted.age"
)

func (s *CoreService) ExportDiagnostics(ctx context.Context, req *ExportDiagnosticsRequest) (*ExportDiagnosticsResponse, error) {
	path, encrypted, err := exportDiagnostics(sWorkingPath, sTempPath, req)
	if err != nil {
		return nil, err
	}
	response := &ExportDiagnosticsResponse{Path: path, Encrypted: encrypted}
	if req.Inline {
		response.Content, err = os.ReadFile(path)
		if err != nil {
			return nil, E.Cause(err, "read the diagnostics zip")
		}
	}
	return response, nil
}

// exportDiagnostics writes the report (see the layout above) and answers its path and whether
// its sensitive part is encrypted.
func exportDiagnostics(workingPath, tempPath string, req *ExportDiagnosticsRequest) (string, bool, error) {
	recipient, err := diagnosticsRecipient()
	if err != nil {
		return "", false, err
	}
	encrypt := recipient != nil && !req.Unencrypted

	stamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	staging := filepath.Join(tempPath, "diagnostics-"+stamp)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", false, E.Cause(err, "create the diagnostics directory")
	}
	defer os.RemoveAll(staging)

	writeGenericFiles(workingPath, staging, req.GenericFiles)
	sensitive := filepath.Join(staging, sensitiveDirectory)
	writeSensitiveFiles(workingPath, sensitive, req.ExtraFiles)
	sensitiveFiles := listFiles(sensitive)
	if encrypt {
		if err := encryptDirectory(sensitive, filepath.Join(staging, encryptedSensitive), recipient); err != nil {
			return "", false, err
		}
	}

	metadata := map[string]any{
		"exportedAt":  time.Now().UTC().Format(time.RFC3339),
		"mode":        sSetupMode.String(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"coreVersion": constants.Version,
		"coreCommit":  constants.Commit(),
		"goVersion":   runtime.Version(),
		"coreState":   static.CoreState.String(),
		"reports":     listCrashReports(workingPath),
		"workingPath": appRelativePath(workingPath),
		"encrypted":   encrypt,
	}
	addAppSummary(metadata, req.ExtraFiles["app-info.json"])
	os.WriteFile(filepath.Join(staging, "metadata.json"), readableJSON(metadata), 0o644)
	os.WriteFile(filepath.Join(staging, "README.txt"), []byte(reportReadme(sensitiveFiles, encrypt)), 0o644)

	destination := req.DestinationPath
	if destination == "" {
		// asked in the clear in a build that encrypts: the copy of the last report, to look inside
		copyOfLast := req.Unencrypted && recipient != nil
		destination = filepath.Join(tempPath, reportName(workingPath, stamp, req.Inline, copyOfLast))
	}
	if err := libbox.CreateZipArchive(staging, destination, false); err != nil {
		return "", false, E.Cause(err, "zip the diagnostics")
	}
	return destination, encrypt, nil
}

// reportNumberFile keeps the number of the last report, in the working directory.
const reportNumberFile = "diagnostics-number"

// reportName names the zip the user saves and sends: hiddify-001.zip, numbered across reports so
// they are told apart in Downloads and in a chat. The unencrypted copy of a report keeps that
// report's number; a zip read back inline is never seen and takes none.
func reportName(workingPath, stamp string, inline, copyOfLast bool) string {
	if inline {
		return "hiddify-" + stamp + ".zip"
	}
	name := fmt.Sprintf("hiddify-%03d", reportNumber(workingPath, !copyOfLast))
	if copyOfLast {
		name += "-unencrypted"
	}
	return name + ".zip"
}

// reportNumber answers the last report's number, or with next (or before any report) a new one,
// which it keeps.
func reportNumber(workingPath string, next bool) int {
	path := filepath.Join(workingPath, reportNumberFile)
	last := 0
	if content, err := os.ReadFile(path); err == nil {
		last, _ = strconv.Atoi(strings.TrimSpace(string(content)))
	}
	if !next && last > 0 {
		return last
	}
	last++
	os.WriteFile(path, []byte(strconv.Itoa(last)), 0o644)
	return last
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
