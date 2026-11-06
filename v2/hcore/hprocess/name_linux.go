package hprocess

import (
	"os"
	"path/filepath"
	"syscall"
)

const renamedEnv = "HIDDIFY_PROCESS_RENAMED"

// SetName gives the process the name process monitors such as htop and top show. The command line
// htop shows comes from argv[0], which only exec sets: the process runs itself again with name and
// args (its arguments without argv[0]), once, keeping its process id and open files. The short
// name (comm, in top and the thread view) is then set on every thread; threads made later take it
// from the one that makes them. Call it again after starting work to name new threads.
func SetName(name string, args []string) {
	if os.Getenv(renamedEnv) == "" {
		if executable, err := os.Executable(); err == nil {
			_ = os.Setenv(renamedEnv, "1")
			// returns only when the exec failed: the process then keeps its file's name
			_ = syscall.Exec(executable, append([]string{name}, args...), os.Environ())
		}
	}
	tasks, _ := filepath.Glob("/proc/self/task/*/comm")
	for _, task := range tasks {
		_ = os.WriteFile(task, []byte(name), 0)
	}
}
