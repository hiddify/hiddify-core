PRODUCT_NAME=libcore
BASENAME=$(PRODUCT_NAME)
BINDIR=bin
LIBNAME=$(PRODUCT_NAME)
SRVNAME=HiddifyService

BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --tags || echo "unknown version")
ifeq ($(OS),Windows_NT)
Not available for Windows! use bash in WSL
endif

TAGS=with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc
IOS_TAGS=with_dhcp,with_low_memory,with_conntrack
GOBUILDLIB=CGO_ENABLED=1 go build -trimpath -tags $(TAGS) -ldflags="-w -s" -buildmode=c-shared
GOBUILDSRV=CGO_ENABLED=1 go build -trimpath 

lib_install:
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.1
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.1

headers:
	go build -buildmode=c-archive -o $(BINDIR)/$(LIBNAME).h ./custom

android: lib_install
	gomobile bind -v -androidapi=21 -javapkg=io.nekohasekai -libname=box -tags=$(TAGS) -trimpath -target=android -o $(BINDIR)/$(LIBNAME).aar github.com/sagernet/sing-box/experimental/libbox ./mobile

ios-full: lib_install
	gomobile bind -v  -target ios,iossimulator,tvos,tvossimulator,macos -libname=box -tags=$(TAGS),with_dhcp,with_low_memory,with_conntrack -trimpath -ldflags="-w -s" -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox ./mobile 
	mv $(BINDIR)/$(PRODUCT_NAME).xcframework $(BINDIR)/$(LIBNAME).xcframework 
	cp Libcore.podspec $(BINDIR)/$(LIBNAME).xcframework/

ios: lib_install
	gomobile bind -v  -target ios -libname=box -tags=$(TAGS),with_dhcp,with_low_memory,with_conntrack -trimpath -ldflags="-w -s" -o $(BINDIR)/Libcore.xcframework github.com/sagernet/sing-box/experimental/libbox ./mobile
	cp Info.plist $(BINDIR)/Libcore.xcframework/



windows-amd64:
	curl http://localhost:18020/exit || echo "exited"
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc $(GOBUILDLIB) -o $(BINDIR)/$(LIBNAME).dll ./custom
	go get github.com/akavel/rsrc
	go install github.com/akavel/rsrc
	cp $(BINDIR)/$(LIBNAME).dll ./$(LIBNAME).dll 
	$$(go env GOPATH)/bin/rsrc -manifest admin_service/cmd/admin_service.manifest -ico ./assets/hiddify-service.ico -o admin_service/cmd/admin_service.syso
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CGO_LDFLAGS="$(LIBNAME).dll" $(GOBUILDSRV) -o $(BINDIR)/$(SRVNAME).exe ./admin_service/cmd
	rm ./$(LIBNAME).dll

linux-amd64:
	env GOOS=linux GOARCH=amd64 $(GOBUILDLIB) -o $(BINDIR)/$(LIBNAME).so ./custom
	cp $(BINDIR)/$(LIBNAME).so ./$(LIBNAME).so
	env GOOS=linux GOARCH=amd64  CGO_LDFLAGS="$(LIBNAME).so" LD_LIBRARY_PATH="./bin/" $(GOBUILDSRV) -o $(BINDIR)/$(SRVNAME) ./admin_service/cmd
	rm ./$(LIBNAME).so
	chmod +x $(BINDIR)/$(SRVNAME)

macos-amd64:
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(IOS_TAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-amd64.dylib ./custom
macos-arm64:
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_ENABLED=1 go build -trimpath -tags $(TAGS),$(IOS_TAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-arm64.dylib ./custom
	
macos-universal: macos-amd64 macos-arm64 
	lipo -create $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME)-arm64.dylib -output $(BINDIR)/$(LIBNAME).dylib
	cp $(BINDIR)/$(LIBNAME).dylib ./$(LIBNAME).dylib 
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="-mmacosx-version-min=10.11" CGO_LDFLAGS="bin/$(LIBNAME).dylib"  CGO_ENABLED=1 $(GOBUILDSRV)  -o $(BINDIR)/$(SRVNAME) ./admin_service/cmd
	rm ./$(LIBNAME).dylib
	chmod +x $(BINDIR)/$(SRVNAME)

clean:
	rm $(BINDIR)/*


release: # Create a new tag for release.
	@echo "previous version was $$(git describe --tags $$(git rev-list --tags --max-count=1))"
	@echo "WARNING: This operation will creates version tag and push to github"
	@bash -c '\
	read -p "Version? (provide the next x.y.z semver) : " TAG && \
	echo $$TAG &&\
	[[ "$$TAG" =~ ^[0-9]{1,2}\.[0-9]{1,2}\.[0-9]{1,2}(\.dev)?$$ ]] || { echo "Incorrect tag. e.g., 1.2.3 or 1.2.3.dev"; exit 1; } && \
	IFS="." read -r -a VERSION_ARRAY <<< "$$TAG" && \
	VERSION_STR="$${VERSION_ARRAY[0]}.$${VERSION_ARRAY[1]}.$${VERSION_ARRAY[2]}" && \
	BUILD_NUMBER=$$(( $${VERSION_ARRAY[0]} * 10000 + $${VERSION_ARRAY[1]} * 100 + $${VERSION_ARRAY[2]} )) && \
	echo "version: $${VERSION_STR}+$${BUILD_NUMBER}" && \
	sed -i -e "s|<key>CFBundleVersion</key>\s*<string>[^<]*</string>|<key>CFBundleVersion</key><string>$${VERSION_STR}</string>|" Info.plist &&\
    sed -i -e "s|<key>CFBundleShortVersionString</key>\s*<string>[^<]*</string>|<key>CFBundleShortVersionString</key><string>$${VERSION_STR}</string>|" Info.plist &&\
	git add Info.plist && \
	git commit -m "release: version $${TAG}" && \
	echo "creating git tag : v$${TAG}" && \
	git push && \
	git tag v$${TAG} && \
	git push -u origin HEAD --tags && \
	echo "Github Actions will detect the new tag and release the new version."'


