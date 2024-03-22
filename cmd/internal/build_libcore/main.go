package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hiddify/hiddify-core/cmd/internal/build_shared"
	_ "github.com/sagernet/gomobile"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/rw"
)

var target string

func init() {
	flag.StringVar(&target, "target", "android", "target platform")
}

func main() {
	flag.Parse()

	switch target {
	case "windows":
		buildWindows()
	case "linux":
		buildLinux()
	case "macos":
		buildMacOS()
	case "android":
		buildAndroid()
	case "ios":
		buildIOS()
	}
}

var (
	sharedFlags []string
	sharedTags  []string
	iosTags     []string
)

const libName = "libcore"

func init() {
	sharedFlags = append(sharedFlags, "-trimpath")
	sharedFlags = append(sharedFlags, "-ldflags", "-s -w")
	sharedTags = append(sharedTags, "with_gvisor", "with_quic", "with_wireguard", "with_ech", "with_utls", "with_clash_api", "with_grpc")
	iosTags = append(iosTags, "with_dhcp", "with_low_memory", "with_conntrack")
}

func setDesktopEnv() {
	os.Setenv("CGO_ENABLED", "1")
	os.Setenv("buildmode", "c-shared")
}

func buildWindows() {
	setDesktopEnv()
	os.Setenv("GOOS", "windows")
	os.Setenv("GOARCH", "amd64")
	os.Setenv("CC", "x86_64-w64-mingw32-gcc")

	args := []string{"build"}
	args = append(args, sharedFlags...)
	args = append(args, "-tags")
	args = append(args, strings.Join(sharedTags, ","))

	output := filepath.Join("bin", libName+".dll")
	args = append(args, "-o", output, "./custom")

	command := exec.Command("go", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func buildLinux() {
	setDesktopEnv()
	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", "amd64")

	args := []string{"build"}
	args = append(args, sharedFlags...)
	args = append(args, "-tags")
	args = append(args, strings.Join(sharedTags, ","))

	output := filepath.Join("bin", libName+".so")
	args = append(args, "-o", output, "./custom")

	command := exec.Command("go", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func buildMacOS() {
	libPaths := []string{}
	for _, arch := range []string{"amd64", "arm64"} {
		out, err := buildMacOSArch(arch)
		if err != nil {
			log.Fatal(err)
			return
		}
		libPaths = append(libPaths, out)
	}

	args := []string{"-create"}
	args = append(args, libPaths...)
	args = append(args, "-output", filepath.Join("bin", libName+".dylib"))

	command := exec.Command("lipo", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func buildMacOSArch(arch string) (string, error) {
	setDesktopEnv()
	os.Setenv("GOOS", "darwin")
	os.Setenv("GOARCH", arch)
	os.Setenv("CGO_CFLAGS", "-mmacosx-version-min=10.11")
	os.Setenv("CGO_LDFLAGS", "-mmacosx-version-min=10.11")

	args := []string{"build"}
	args = append(args, sharedFlags...)
	tags := append(sharedTags, iosTags...)
	args = append(args, "-tags")
	args = append(args, strings.Join(tags, ","))

	filename := libName + "-" + arch + ".dylib"
	output := filepath.Join("bin", filename)
	args = append(args, "-o", output, "./custom")

	command := exec.Command("go", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		return "", err
	}
	return output, nil
}

func buildAndroid() {
	build_shared.FindMobile()
	build_shared.FindSDK()

	args := []string{
		"bind",
		"-v",
		"-androidapi", "21",
		"-javapkg=io.nekohasekai",
		"-libname=box",
		"-target=android",
	}

	args = append(args, sharedFlags...)
	args = append(args, "-tags")
	args = append(args, strings.Join(sharedTags, ","))

	output := filepath.Join("bin", libName+".aar")
	args = append(args, "-o", output, "github.com/sagernet/sing-box/experimental/libbox", "./mobile")

	command := exec.Command(build_shared.GoBinPath+"/gomobile", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func buildIOS() {
	build_shared.FindMobile()

	args := []string{
		"bind",
		"-v",
		"-libname=box",
		"-target", "ios,iossimulator,tvos,tvossimulator,macos",
	}

	args = append(args, sharedFlags...)
	tags := append(sharedTags, iosTags...)
	args = append(args, "-tags")
	args = append(args, strings.Join(tags, ","))

	output := filepath.Join("bin", "Libcore.xcframework")
	args = append(args, "-o", output, "github.com/sagernet/sing-box/experimental/libbox", "./mobile")

	command := exec.Command(build_shared.GoBinPath+"/gomobile", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Debug("command: ", command.String())
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}

	rw.CopyFile("Info.plist", filepath.Join(output, "Info.plist"))
}
