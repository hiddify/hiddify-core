// HiddifyAdmin.exe: the one Hiddify binary that asks for administrator rights.
//
// The core launches it (UAC prompt) to install/start the VPN background service.
// It forwards its command line to parseCli() in hiddify-core.dll like
// HiddifyCli.exe, but it is a GUI-subsystem program with a requireAdministrator
// manifest, an icon and a VERSIONINFO, so the UAC dialog shows "Hiddify" with the
// app icon instead of a console tool, and no console window flashes while elevated.
// Written in C for the same reason as cmd/clilauncher (ML antivirus engines flag
// small Go stubs).
#include <windows.h>
#include <shellapi.h>
#include <stdlib.h>

__declspec(dllimport) char *parseCli(int argc, char **argv);

static char *to_utf8(const wchar_t *w) {
  int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
  char *s = (char *)malloc(n > 0 ? (size_t)n : 1);
  if (!s) return NULL;
  if (n > 0) WideCharToMultiByte(CP_UTF8, 0, w, -1, s, n, NULL, NULL); else s[0] = 0;
  return s;
}

int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, PWSTR pCmdLine, int nCmdShow) {
  (void)hInstance; (void)hPrevInstance; (void)pCmdLine; (void)nCmdShow;
  int argc = 0;
  wchar_t **wargv = CommandLineToArgvW(GetCommandLineW(), &argc);
  if (!wargv || argc < 2) return 2;  // nothing to do without a command
  char **argv = (char **)calloc((size_t)argc + 1, sizeof(char *));
  if (!argv) return 1;
  for (int i = 0; i < argc; i++) {
    argv[i] = to_utf8(wargv[i]);
    if (!argv[i]) return 1;
  }
  LocalFree(wargv);
  const char *result = parseCli(argc, argv);
  int rc = (result && result[0]) ? 1 : 0;  // parseCli returns "" on success, else the error text
  for (int i = 0; i < argc; i++) free(argv[i]);
  free(argv);
  return rc;
}
