// The launchers of hiddify-core: each forwards its command line to parseCli() in the core library.
// One source, built four ways:
//
//   Windows (windows-amd64 in the Makefile)
//   HiddifyCli.exe        console tool, prints parseCli's result
//   HiddifyAdmin.exe      -DHIDDIFY_LAUNCHER_GUI: GUI subsystem, with a requireAdministrator
//                         manifest, so the UAC dialog shows "Hiddify" and the app icon and no
//                         console window flashes; it installs the VPN service
//   core\Hiddify.exe      -DHIDDIFY_LAUNCHER_GUI -DHIDDIFY_LAUNCHER_LIBRARY=...: the app's core in
//                         a process of its own (`core` command), named Hiddify with the core icon;
//                         it sits in a subfolder, beside the app's own Hiddify.exe, so it loads
//                         hiddify-core.dll (and libcronet.dll) from the folder above
//
//   macOS (the app's macos/scripts/embed-core-helper.sh)
//   Hiddify Core.app      -DHIDDIFY_LAUNCHER_LIBRARY=... -DHIDDIFY_LAUNCHER_MACOS_APP: the same core
//                         process, in a helper bundle inside Hiddify.app, which gives it the name
//                         Hiddify and the core icon; it loads hiddify-core.dylib from the app's
//                         Frameworks
//
// HIDDIFY_LAUNCHER_LIBRARY is the library's path relative to the launcher's folder; without it the
// library is linked (Windows, beside the launcher).
//
// Written in C on purpose: a ~20 KB C program with one import is as plain as a PE file gets. The Go
// stub it replaced was flagged by machine-learning antivirus engines (Bkav, McAfee, DeepInstinct)
// whatever link flags were used, because small Go executables share byte-level features with
// Go-based malware loaders.
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef char *(*parse_cli_fn)(int argc, char **argv);

#ifdef _WIN32
#ifndef UNICODE
#define UNICODE
#endif
#include <windows.h>
#include <shellapi.h>

#define WIDEN2(x) L##x
#define WIDEN(x) WIDEN2(x)

#ifdef HIDDIFY_LAUNCHER_LIBRARY
// Loads the library at HIDDIFY_LAUNCHER_LIBRARY, relative to this executable's folder. The
// library's folder is also added to the DLL search path, for libcronet.dll, which the core loads
// when it needs it.
static parse_cli_fn load_parse_cli(void) {
  wchar_t path[MAX_PATH * 2];
  wchar_t full[MAX_PATH * 2];
  DWORD n = GetModuleFileNameW(NULL, path, MAX_PATH * 2);
  if (n == 0 || n >= MAX_PATH * 2) return NULL;
  wchar_t *slash = wcsrchr(path, L'\\');
  if (!slash) return NULL;
  slash[1] = 0;
  if (wcslen(path) + wcslen(WIDEN(HIDDIFY_LAUNCHER_LIBRARY)) >= MAX_PATH * 2) return NULL;
  wcscat(path, WIDEN(HIDDIFY_LAUNCHER_LIBRARY));
  if (!GetFullPathNameW(path, MAX_PATH * 2, full, NULL)) return NULL;
  wcscpy(path, full);
  slash = wcsrchr(path, L'\\');
  if (!slash) return NULL;
  *slash = 0;
  SetDllDirectoryW(path);
  HMODULE core = LoadLibraryExW(full, NULL, LOAD_WITH_ALTERED_SEARCH_PATH);
  if (!core) return NULL;
  return (parse_cli_fn)(void *)GetProcAddress(core, "parseCli");
}
#else
__declspec(dllimport) char *parseCli(int argc, char **argv);
static parse_cli_fn load_parse_cli(void) { return parseCli; }
#endif

static char *to_utf8(const wchar_t *w) {
  int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
  char *s = (char *)malloc(n > 0 ? (size_t)n : 1);
  if (!s) return NULL;
  if (n > 0) WideCharToMultiByte(CP_UTF8, 0, w, -1, s, n, NULL, NULL); else s[0] = 0;
  return s;
}

// Go's os.Args are UTF-8: the command line is read again as wide characters and converted.
static char **utf8_arguments(int *argc) {
  wchar_t **wargv = CommandLineToArgvW(GetCommandLineW(), argc);
  if (!wargv || *argc < 1) return NULL;
  char **argv = (char **)calloc((size_t)*argc + 1, sizeof(char *));
  if (!argv) return NULL;
  for (int i = 0; i < *argc; i++) {
    argv[i] = to_utf8(wargv[i]);
    if (!argv[i]) return NULL;
  }
  LocalFree(wargv);
  return argv;
}

#else  // macOS
#include <dlfcn.h>
#include <limits.h>
#include <mach-o/dyld.h>

#ifndef HIDDIFY_LAUNCHER_LIBRARY
#error "HIDDIFY_LAUNCHER_LIBRARY must name the core library"
#endif

static parse_cli_fn load_parse_cli(void) {
  char executable[PATH_MAX];
  char path[PATH_MAX * 2];
  char resolved[PATH_MAX];
  uint32_t size = sizeof(executable);
  if (_NSGetExecutablePath(executable, &size) != 0) return NULL;
  char *slash = strrchr(executable, '/');
  if (!slash) return NULL;
  slash[1] = 0;
  if (snprintf(path, sizeof(path), "%s%s", executable, HIDDIFY_LAUNCHER_LIBRARY) >= (int)sizeof(path)) return NULL;
  if (!realpath(path, resolved)) return NULL;
  void *core = dlopen(resolved, RTLD_NOW);
  if (!core) {
    fprintf(stderr, "%s\n", dlerror());
    return NULL;
  }
  return (parse_cli_fn)dlsym(core, "parseCli");
}

#ifdef HIDDIFY_LAUNCHER_MACOS_APP
#include <CoreFoundation/CoreFoundation.h>
#include <objc/message.h>
#include <objc/runtime.h>

// NSActivityUserInitiatedAllowingIdleSystemSleep: user-initiated work, which keeps App Nap away
// but lets the Mac sleep when idle.
#define HIDDIFY_ACTIVITY_OPTIONS (0x00FFFFFFULL & ~(1ULL << 20))

// Makes the process a background-only app: Activity Monitor shows its bundle's name and icon only
// for apps. An app can be put in App Nap, which throttles its priority and I/O (hiddify-app#2343),
// so it holds a user-initiated activity for its whole life, as Apple documents for that case.
static void become_background_app(void) {
  id (*send)(id, SEL) = (id (*)(id, SEL))objc_msgSend;
  id (*send_activity)(id, SEL, unsigned long long, id) =
      (id (*)(id, SEL, unsigned long long, id))objc_msgSend;
  Class application = objc_getClass("NSApplication");
  Class process_info = objc_getClass("NSProcessInfo");
  if (!application || !process_info) return;
  send((id)application, sel_registerName("sharedApplication"));
  id info = send((id)process_info, sel_registerName("processInfo"));
  id activity = send_activity(info, sel_registerName("beginActivityWithOptions:reason:"),
                              HIDDIFY_ACTIVITY_OPTIONS, (id)CFSTR("Hiddify core"));
  // kept for the life of the process
  send(activity, sel_registerName("retain"));
}
#endif
#endif

static int run(int argc, char **argv) {
  parse_cli_fn parse_cli = load_parse_cli();
  if (!parse_cli) return 3;
  if (!argv) return 1;
#ifdef HIDDIFY_LAUNCHER_GUI
  if (argc < 2) return 2;  // nothing to do without a command
#endif
  const char *result = parse_cli(argc, argv);
#if defined(_WIN32) && !defined(HIDDIFY_LAUNCHER_GUI)
  if (result) {
    fputs(result, stdout);
    fputc('\n', stdout);
    fflush(stdout);
  }
  return 0;
#else
  return (result && result[0]) ? 1 : 0;  // parseCli returns "" on success, else the error text
#endif
}

#if defined(_WIN32) && defined(HIDDIFY_LAUNCHER_GUI)
int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, PWSTR pCmdLine, int nCmdShow) {
  (void)hInstance; (void)hPrevInstance; (void)pCmdLine; (void)nCmdShow;
  int argc = 0;
  char **argv = utf8_arguments(&argc);
  return run(argc, argv);
}
#elif defined(_WIN32)
int main(void) {
  int argc = 0;
  char **argv = utf8_arguments(&argc);
  return run(argc, argv);
}
#else
int main(int argc, char **argv) {
#ifdef HIDDIFY_LAUNCHER_MACOS_APP
  become_background_app();
#endif
  return run(argc, argv);
}
#endif
