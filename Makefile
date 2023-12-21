PRODUCT_NAME=libcore
BASENAME=hiddify-$(PRODUCT_NAME)
BINDIR=bin

BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --tags || echo "unknown version")
NAME=$(BASENAME)-$@

TAGS=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc
IOS_TAGS=with_dhcp,with_low_memory,with_conntrack
GOBUILD=CGO_ENABLED=1 go build -trimpath -tags $(TAGS) -ldflags="-w -s" -buildmode=c-shared

lib_install:
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.1
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.1

headers:
	go build -buildmode=c-archive -o $(BINDIR)/$(NAME).h ./custom

android: lib_install
	gomobile bind -v -androidapi=21 -javapkg=io.nekohasekai -libname=box -tags=$(TAGS) -trimpath -target=android -o $(BINDIR)/$(NAME).aar github.com/sagernet/sing-box/experimental/libbox ./mobile

ios-full: lib_install
	gomobile bind -v  -target ios,iossimulator,tvos,tvossimulator,macos -libname=box -tags=$(TAGS),with_dhcp,with_low_memory,with_conntrack -trimpath -ldflags="-w -s" -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox ./mobile
ios: lib_install
	gomobile bind -v  -target ios -libname=box -tags=$(TAGS),with_dhcp,with_low_memory,with_conntrack -trimpath -ldflags="-w -s" -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox ./mobile &&\
	mv $(BINDIR)/$(PRODUCT_NAME).xcframework $(BINDIR)/$(NAME).xcframework


windows-amd64:
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GOBUILD) -o $(BINDIR)/$(NAME).dll ./custom

linux-amd64:
	env GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINDIR)/$(NAME).so ./custom

macos-amd64:
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(IOS_TAGS) -buildmode=c-shared -o $(BINDIR)/$(NAME).dylib ./custom
macos-arm64:
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(IOS_TAGS) -buildmode=c-shared -o $(BINDIR)/$(NAME).dylib ./custom
macos-universal: macos-amd64 macos-arm64 
	lipo -create $(BINDIR)/$(BASENAME)-macos-amd64.dylib $(BINDIR)/$(BASENAME)-macos-arm64.dylib -output $(BINDIR)/$(NAME).dylib

clean:
	rm $(BINDIR)/*
