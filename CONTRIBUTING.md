Hiddify uses [Go](https://go.dev), make sure that you have the correct version installed before starting development. You can use the following commands to check your installed version:


```shell
$ go version

# example response
go version go1.21.1 darwin/arm64
```

### Working with the Go Code

> if you're not interested in building/contributing to the Go code, you can skip this section

The Go code for Hiddify can be found in the `hiddify-core` folder, as a [git submodule](https://git-scm.com/book/en/v2/Git-Tools-Submodules) and in [core repository](https://github.com/hiddify/hiddify-next-core). The entrypoints for the desktop version are available in the [`hiddify-core/custom`](https://github.com/hiddify/hiddify-next-core/tree/main/custom) folder and for the mobile version they can be found in the [`hiddify-core/mobile`](https://github.com/hiddify/hiddify-next-core/tree/main/mobile) folder.

For the desktop version, we have to compile the Go code into a C shared library. We are providing a Makefile to generate the C shared libraries for all operating systems. The following Make commands will build hiddify-core and copy the resulting output in [`hiddify-core/bin`](https://github.com/hiddify/hiddify-next-core/tree/main/bin):

- `make windows-amd64`
- `make linux-amd64`
- `make macos-universal`

For the mobile version, we are using the [`gomobile`](https://github.com/golang/go/wiki/Mobile) tools. The following Make commands will build hiddify-core for Android and iOS and copy the resulting output in [`hiddify-core/bin`](https://github.com/hiddify/hiddify-next-core/tree/main/bin):

- `make android`
- `make ios`
