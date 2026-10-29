// HiddifyCli.exe launcher: forwards the command line to parseCli() exported by
// hiddify-core.dll (same contract as the former Go stub in cmd/bydll).
//
// Written in C on purpose: a ~20 KB C program with one import is as plain as a
// PE file gets. The Go stub it replaces was flagged by machine-learning antivirus
// engines (Bkav, McAfee, DeepInstinct) whatever link flags were used, because
// small Go executables share byte-level features with Go-based malware loaders.
#ifndef UNICODE
#define UNICODE
#endif
#include <windows.h>
#include <shellapi.h>
#include <stdio.h>
#include <stdlib.h>

__declspec(dllimport) char *parseCli(int argc, char **argv);

static char *to_utf8(const wchar_t *w) {
  int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL);
  char *s = (char *)malloc(n > 0 ? (size_t)n : 1);
  if (!s) return NULL;
  if (n > 0) WideCharToMultiByte(CP_UTF8, 0, w, -1, s, n, NULL, NULL); else s[0] = 0;
  return s;
}

int main(void) {
  int argc = 0;
  wchar_t **wargv = CommandLineToArgvW(GetCommandLineW(), &argc);
  if (!wargv || argc < 1) return 1;
  char **argv = (char **)calloc((size_t)argc + 1, sizeof(char *));
  if (!argv) return 1;
  for (int i = 0; i < argc; i++) {
    argv[i] = to_utf8(wargv[i]);  // Go's os.Args are UTF-8
    if (!argv[i]) return 1;
  }
  LocalFree(wargv);
  const char *result = parseCli(argc, argv);
  if (result) {
    fputs(result, stdout);
    fputc('\n', stdout);
    fflush(stdout);
  }
  for (int i = 0; i < argc; i++) free(argv[i]);
  free(argv);
  return 0;
}
