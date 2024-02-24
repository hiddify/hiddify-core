//go:build windows

package config

// "golang.org/x/sys/windows"

func ExecuteCmd(exe string, args string, background bool) (string, error) {
	// verb := "runas"
	// cwd, _ := os.Getwd()

	// verbPtr, _ := syscall.UTF16PtrFromString(verb)
	// exePtr, _ := syscall.UTF16PtrFromString(exe)
	// cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	// argPtr, _ := syscall.UTF16PtrFromString(args)

	// var showCmd int32 = 1 //SW_NORMAL

	// err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	// if err != nil {
	// 	return "", err
	// }
	return "", nil
}
