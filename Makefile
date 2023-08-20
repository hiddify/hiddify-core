NAME=hiddify-libcore
BINDIR=bin

TAGS=with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api
GOBUILD=CGO_ENABLED=1 go build -trimpath -tags $(TAGS) -ldflags="-w -s" -buildmode=c-shared

lib_install:
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.0.0-20230728014906-3de089147f59
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.0.0-20230728014906-3de089147f59

headers:
	go build -buildmode=c-archive -o $(BINDIR)/$(NAME)-$@.h ./custom

android:
	gomobile bind -v -androidapi=21 -javapkg=io.nekohasekai -libname=box -tags=$(TAGS) -trimpath -ldflags="-w -s" -target=android -o $(BINDIR)/$(NAME)-$@.aar github.com/sagernet/sing-box/experimental/libbox ./mobile

windows-amd64:
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.dll ./custom

windows-386:
	env GOOS=windows GOARCH=386 CC=i686-w64-mingw32-gcc $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.dll ./custom

linux-amd64:
	env GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.so ./custom

linux-386:
	env GOOS=linux GOARCH=386  $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.so ./custom

macos-amd64:
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags with_gvisor,with_lwip -buildmode=c-shared -o $(BINDIR)/$(NAME)-$@.dylib ./custom
macos-arm64:
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags with_gvisor,with_lwip -buildmode=c-shared -o $(BINDIR)/$(NAME)-$@.dylib ./custom

macos-universal: macos-amd64 macos-arm64 
	lipo -create $(BINDIR)/$(NAME)-macos-amd64.dylib $(BINDIR)/$(NAME)-macos-arm64.dylib -output $(BINDIR)/$(NAME)-$@.dylib
	

clean:
	rm $(BINDIR)/*