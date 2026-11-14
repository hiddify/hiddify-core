package hcore

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestListCrashReportsTellsCrashesFromOtherOutput(t *testing.T) {
	working := t.TempDir()
	writeTestFile(t, filepath.Join(working, "crash_reports", "2026-10-30T10-00-00", "go.log"), "panic: boom\n\ngoroutine 1 [running]:\nmain.main()\n")
	writeTestFile(t, filepath.Join(working, "crash_reports", "2026-10-31T10-00-00", "go.log"), "some library warning\n")
	writeTestFile(t, filepath.Join(working, "oom_reports", "2026-10-29T10-00-00", "metadata.json"), "{}")
	writeTestFile(t, filepath.Join(working, "data", "stderrGRPC_BACKGROUND.log"), "fatal error: concurrent map writes\n")
	writeTestFile(t, filepath.Join(working, "data", "stderrGRPC_NORMAL.log"), "   \n")

	reports := listCrashReports(working)
	byName := map[string]bool{}
	for _, report := range reports {
		byName[report.Name] = report.IsCrash
	}
	require.Equal(t, map[string]bool{
		"2026-10-29T10-00-00":            true,  // an OOM kill is a crash
		"2026-10-30T10-00-00":            true,  // a panic
		"2026-10-31T10-00-00":            false, // other stderr output
		"data/stderrGRPC_BACKGROUND.log": true,  // the background core's live file
	}, byName)
	require.Equal(t, "2026-10-29T10:00:00Z", reports[0].CrashedAt)
	require.Equal(t, "oom_reports", reports[0].Source)
}

func TestExportDiagnosticsZipsSanitizedFiles(t *testing.T) {
	working, temp := t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(working, "crash_reports", "2026-10-30T10-00-00", "go.log"), "panic: boom at 1.2.3.4\n")
	writeTestFile(t, filepath.Join(working, "crash_reports", "2026-10-30T10-00-00", "metadata.json"), `{"crashedAt":"2026-10-30T10:00:00Z"}`)
	writeTestFile(t, filepath.Join(working, "data", "stderrGRPC_BACKGROUND.log"), "fatal error: x\n")
	writeTestFile(t, filepath.Join(working, "data", "box.log"), strings.Repeat("INFO connection to vless://user@5.6.7.8:443\n", 3))
	writeTestFile(t, filepath.Join(working, "data", "goroutines-grpc_background.txt"), "goroutine 7 [select]:\nlast run\n")
	keepPreviousGoroutineSnapshot(working, SetupMode_GRPC_BACKGROUND)
	require.NoFileExists(t, filepath.Join(working, "data", "goroutines-grpc_background.txt"))
	writeTestFile(t, filepath.Join(working, "data", "current-config.json"), `{"outbounds":[{"type":"vless","tag":"p","server":"my.server.example","server_port":443,"uuid":"0a1b2c3d-1111-2222-3333-444455556666"}]}`)

	response, encrypted, err := exportDiagnostics(working, temp, &ExportDiagnosticsRequest{ExtraFiles: map[string][]byte{"app-logs.txt": []byte("app log line password=hunter2\n")}})
	require.NoError(t, err)
	require.False(t, encrypted, "a build without a key")
	require.FileExists(t, response)
	entries, _ := os.ReadDir(temp)
	require.Len(t, entries, 1, "the staging directory is removed")

	reader, err := zip.OpenReader(response)
	require.NoError(t, err)
	defer reader.Close()
	files := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(opened)
		opened.Close()
		require.NoError(t, err)
		files[strings.TrimPrefix(file.Name, strings.SplitN(file.Name, "/", 2)[0]+"/")] = string(content)
	}
	for _, name := range []string{"crash_reports/2026-10-30T10-00-00/go.log", "crash_reports/2026-10-30T10-00-00/metadata.json", "sensitive/logs/data-stderrGRPC_BACKGROUND.log", "sensitive/logs/box.log", "sensitive/config.json", "goroutines/goroutines-grpc_background.previous.txt", "sensitive/app/app-logs.txt", "metadata.json"} {
		require.Contains(t, files, name, name)
	}
	for name := range files {
		// only the generic files are outside the part that is encrypted
		require.True(t, name == "metadata.json" || name == "README.txt" || strings.HasPrefix(name, "goroutines/") || strings.HasPrefix(name, "crash_reports/") || strings.HasPrefix(name, "app/") || strings.HasPrefix(name, "sensitive/"), name)
	}
	require.NotContains(t, files["crash_reports/2026-10-30T10-00-00/go.log"], "1.2.3.4", "text is scrubbed")
	require.NotContains(t, files["sensitive/logs/box.log"], "5.6.7.8")
	require.NotContains(t, files["sensitive/config.json"], "my.server.example")
	require.NotContains(t, files["sensitive/config.json"], "0a1b2c3d")
	require.Contains(t, files["sensitive/config.json"], `"vless"`)
	require.Contains(t, files["sensitive/config.json"], `"server_port": 443`)
	require.NotContains(t, files["sensitive/app/app-logs.txt"], "hunter2")
	require.Contains(t, files["goroutines/goroutines-grpc_background.previous.txt"], "last run")
	require.NotContains(t, strings.Join(slices.Collect(maps.Keys(files)), " "), "goroutines-old", "not this process's goroutines")
	require.Contains(t, files["metadata.json"], `"mode"`)
}

// The snapshot is the goroutine profile, not runtime.Stack(all): see goroutineDump.
func TestGoroutineDumpHasEveryStack(t *testing.T) {
	parked := make(chan struct{})
	defer close(parked)
	go func() { <-parked }()
	dump := goroutineDump()
	require.Contains(t, dump, "goroutine profile: total ")
	require.Contains(t, dump, "hcore.TestGoroutineDumpHasEveryStack")
	require.Contains(t, dump, "diagnostics_test.go:")
}

func TestAppRelativePath(t *testing.T) {
	require.Equal(t, "apple.hiddify.com", appRelativePath("/Users/someone/Library/Application Support/apple.hiddify.com"))
	require.Equal(t, "app.hiddify.com/files", appRelativePath("/data/user/0/app.hiddify.com/files"))
	require.Equal(t, "Hiddify/portable_data", appRelativePath("/home/someone/apps/Hiddify/portable_data"))
	require.Equal(t, "data", appRelativePath("/home/someone/data"))
}

func TestAppSummaryInMetadata(t *testing.T) {
	metadata := map[string]any{"os": "darwin"}
	addAppSummary(metadata, []byte(`{"app":{"version":"5.9.1"},"system":{"os":"macos"},"settings":{"x":1},"proxies":[1,2],"os":"no"}`))
	content, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.JSONEq(t, `{"os":"darwin","app":{"version":"5.9.1"},"system":{"os":"macos"},"settings":{"x":1}}`, string(content))
	addAppSummary(metadata, []byte("not json"))
	require.Len(t, metadata, 4)
}

// Setup names the stderr file for the mode (grpc_server.go): diagnostics must look for that name.
func TestLiveCrashFilesHaveSetupStderrNames(t *testing.T) {
	require.Contains(t, liveCrashFiles, "data/stderrGRPC_NORMAL.log")
	require.Contains(t, liveCrashFiles, "data/stderrGRPC_BACKGROUND_INSECURE.log")
	require.NotContains(t, liveCrashFiles, "data/stderr1.log")
}

// zipFiles is every file of the zip in content, by name without the root folder.
func zipFiles(t *testing.T, content []byte) map[string][]byte {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		opened, err := file.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(opened)
		opened.Close()
		require.NoError(t, err)
		files[strings.SplitN(file.Name, "/", 2)[1]] = data
	}
	return files
}

func TestExportDiagnosticsEncryptsTheSensitivePart(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	previous := diagnosticsPublicKey
	diagnosticsPublicKey = identity.Recipient().String()
	defer func() { diagnosticsPublicKey = previous }()

	working, temp := t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(working, "data", "box.log"), "INFO a secret log line from 5.6.7.8\n")
	appInfo := []byte(`{"app":{"version":"5.9.3"},"settings":{"mode":"tun","window_size":"<set>"}}`)

	writeTestFile(t, filepath.Join(working, "data", "goroutines-grpc_normal.txt"), "goroutine 1 [running]:\nmain.main()\n")
	keepPreviousGoroutineSnapshot(working, SetupMode_GRPC_NORMAL)
	appCrash := []byte("StateError: boom\n#0 main (package:hiddify/main.dart:10)\n")
	path, encrypted, err := exportDiagnostics(working, temp, &ExportDiagnosticsRequest{
		ExtraFiles:   map[string][]byte{"app-info.json": appInfo},
		GenericFiles: map[string][]byte{"app-crash-1.txt": appCrash},
	})
	require.NoError(t, err)
	require.True(t, encrypted)
	require.True(t, strings.HasSuffix(path, ".zip") && !strings.HasSuffix(path, "-unencrypted.zip"), path)
	report, err := os.ReadFile(path)
	require.NoError(t, err)
	files := zipFiles(t, report)
	require.ElementsMatch(t, []string{"metadata.json", "README.txt", "sensitive.encrypted.age", "goroutines/goroutines-grpc_normal.previous.txt", "app/app-crash-1.txt"}, slices.Collect(maps.Keys(files)))
	require.Equal(t, appCrash, files["app/app-crash-1.txt"], "the app's exception stays readable")
	require.NotContains(t, string(report), "secret log line")
	require.Contains(t, string(files["metadata.json"]), `"version": "5.9.3"`)
	require.Contains(t, string(files["metadata.json"]), `"encrypted": true`)
	require.Contains(t, string(files["metadata.json"]), `"window_size": "<set>"`, "plain UTF-8, not \\u003c")
	require.Contains(t, string(files["README.txt"]), "    logs/box.log\n")

	plain, err := age.Decrypt(bytes.NewReader(files["sensitive.encrypted.age"]), identity)
	require.NoError(t, err)
	content, err := io.ReadAll(plain)
	require.NoError(t, err)
	sensitive := zipFiles(t, content)
	require.Contains(t, string(sensitive["logs/box.log"]), "secret log line from <ipv4>", "scrubbed, then encrypted")
	require.Equal(t, appInfo, sensitive["app/app-info.json"])

	// asked unencrypted, for the user to look at
	path, encrypted, err = exportDiagnostics(working, temp, &ExportDiagnosticsRequest{Unencrypted: true})
	require.NoError(t, err)
	require.False(t, encrypted)
	require.True(t, strings.HasSuffix(path, "-unencrypted.zip"), path)
	report, _ = os.ReadFile(path)
	files = zipFiles(t, report)
	require.Contains(t, files, "sensitive/logs/box.log")
	require.Contains(t, string(files["README.txt"]), "NOT encrypted")
}

func TestExportDiagnosticsRefusesAKeyThatIsNotAge(t *testing.T) {
	previous := diagnosticsPublicKey
	diagnosticsPublicKey = "not-an-age-key"
	defer func() { diagnosticsPublicKey = previous }()
	_, _, err := exportDiagnostics(t.TempDir(), t.TempDir(), &ExportDiagnosticsRequest{})
	require.Error(t, err)
}
