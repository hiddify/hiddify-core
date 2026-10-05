.ONESHELL:
PRODUCT_NAME=hiddify-core
BASENAME=$(PRODUCT_NAME)
BINDIR=bin
LIBNAME=$(PRODUCT_NAME)
CLINAME=HiddifyCli

BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --tags || echo "unknown version")
ifeq ($(OS),Windows_NT)
Not available for Windows! use bash in WSL
endif
CRONET_GO_VERSION := $(shell cat hiddify-sing-box/.github/CRONET_GO_VERSION)
TAGS=with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_grpc,with_awg,tfogo_checklinkname0,with_naive_outbound,with_conntrack,with_embedded_tor,with_openvpn,with_openconnect
IOS_ADD_TAGS=with_dhcp,with_low_memory,with_purego
MACOS_ADD_TAGS=with_dhcp
WINDOWS_ADD_TAGS=with_purego
# Optional build-time secrets (GitHub Actions): HIDDIFY_SECRET_KEY (base64 AES-256 key) is embedded,
# and PSIPHON_CONFIG (Psiphon config, JSON or base64) is embedded encrypted with it (config "hiddify").
EMBED_LDFLAGS=
ifneq ($(strip $(HIDDIFY_SECRET_KEY)),)
EMBED_LDFLAGS+=-X github.com/sagernet/sing-box/hiddify/secret.Key=$(strip $(HIDDIFY_SECRET_KEY))
ifneq ($(strip $(PSIPHON_CONFIG)),)
PSIPHON_CONFIG_ENCRYPTED:=$(shell go run ./cmd/internal/encryptconfig)
ifeq ($(PSIPHON_CONFIG_ENCRYPTED),)
$(error failed to encrypt PSIPHON_CONFIG)
endif
EMBED_LDFLAGS+=-X github.com/sagernet/sing-box/protocol/psiphon.embeddedHiddifyConfig=$(PSIPHON_CONFIG_ENCRYPTED)
endif
endif
LDFLAGS=-w -s -checklinkname=0 -buildid= $${CODE_VERSION} $(EMBED_LDFLAGS)
GOBUILDLIB=CGO_ENABLED=1 go build -trimpath -ldflags="$(LDFLAGS)" -buildmode=c-shared
GOBUILDSRV=CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -trimpath -tags $(TAGS)

CRONET_DIR=./cronet
.PHONY: protos
protos:
	go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
	# protoc --go_out=./ --go-grpc_out=./ --proto_path=hiddifyrpc hiddifyrpc/*.proto
	# for f in $(shell find v2 -name "*.proto"); do \
	# 	protoc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=./ --go-grpc_out=./  $$f; \
	# done
	# for f in $(shell find extension -name "*.proto"); do \
	# 	protoc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=./ --go-grpc_out=./  $$f; \
	# done
	protoc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=./ --go-grpc_out=./  $(shell find v2 -name "*.proto") $(shell find extension -name "*.proto")
	protoc --doc_out=./docs  --doc_opt=markdown,hiddifyrpc.md $(shell find v2 -name "*.proto") $(shell find extension -name "*.proto")
	# protoc --js_out=import_style=commonjs,binary:./extension/html/rpc/ --grpc-web_out=import_style=commonjs,mode=grpcwebtext:./extension/html/rpc/ $(shell find v2 -name "*.proto") $(shell find extension -name "*.proto")
	# npx browserify extension/html/rpc/extension.js >extension/html/rpc.js


lib_install: prepare
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.11
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.11
	npm install

headers:
	go build -buildmode=c-archive -o $(BINDIR)/ ./platform/desktop2

android: lib_install
	CGO_LDFLAGS="-O2 -g -s -w -Wl,-z,max-page-size=16384" gomobile bind -v -androidapi=21 -javapkg=com.hiddify.core -libname=hiddify-core -tags=$(TAGS) -trimpath -ldflags="$(LDFLAGS)" -target=android -gcflags "all=-N -l" -o $(BINDIR)/$(LIBNAME).aar github.com/sagernet/sing-box/experimental/libbox ./platform/mobile

ios-full: lib_install
	gomobile bind -v  -target ios,iossimulator,tvos,tvossimulator,macos -libname=hiddify-core -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(BINDIR)/$(PRODUCT_NAME).xcframework github.com/sagernet/sing-box/experimental/libbox ./platform/mobile 
	mv $(BINDIR)/$(PRODUCT_NAME).xcframework $(BINDIR)/$(LIBNAME).xcframework 
	cp HiddifyCore.podspec $(BINDIR)/$(LIBNAME).xcframework/

ios: lib_install
	gomobile bind -v  -target ios -libname=hiddify-core -tags=$(TAGS),$(IOS_ADD_TAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(BINDIR)/HiddifyCore.xcframework github.com/sagernet/sing-box/experimental/libbox ./platform/mobile
	cp Info.plist $(BINDIR)/HiddifyCore.xcframework/


webui:
	curl -L -o webui.zip  https://github.com/hiddify/Yacd-meta/archive/gh-pages.zip 
	unzip -d ./ -q webui.zip
	rm webui.zip
	rm -rf bin/webui
	mv Yacd-meta-gh-pages bin/webui

.PHONY: build
windows-amd64: prepare
	rm -rf $(BINDIR)/*
	$(MAKE) BINDIR=$(BINDIR) fetch-libcronet-windows_amd64.dll
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc  $(GOBUILDLIB) -tags $(TAGS),$(WINDOWS_ADD_TAGS)   -o $(BINDIR)/$(LIBNAME).dll ./platform/desktop
	echo "core built, now building cli" 
	ls -R $(BINDIR)/
	go install -mod=readonly github.com/akavel/rsrc@latest ||echo "rsrc error in installation"
	go run ./cli tunnel exit
	cp $(BINDIR)/$(LIBNAME).dll ./$(LIBNAME).dll
	$$(go env GOPATH)/bin/rsrc -ico ./assets/hiddify-cli.ico -o ./cmd/bydll/cli.syso ||echo "rsrc error in syso"
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CGO_LDFLAGS="$(LIBNAME).dll" $(GOBUILDSRV) -o $(BINDIR)/$(CLINAME).exe ./cmd/bydll
	rm ./*.dll
	if [ ! -f $(BINDIR)/$(LIBNAME).dll -o ! -f $(BINDIR)/$(CLINAME).exe -o ! -f $(BINDIR)/libcronet.dll ]; then \
		echo "Error: $(LIBNAME).dll, $(CLINAME).exe or libcronet.dll missing"; \
		exit 1; \
	fi

# 	make webui
	



# Prebuilt libcronet from cronet-go's lib/<os>_<arch>/ (same as upstream sing-box CI).
# Usage: make fetch-libcronet-windows_amd64.dll
fetch-libcronet-%:
	rm -rf $(BINDIR)/.cronet-go-lib
	git init -q $(BINDIR)/.cronet-go-lib
	git -C $(BINDIR)/.cronet-go-lib remote add origin https://github.com/sagernet/cronet-go.git
	git -C $(BINDIR)/.cronet-go-lib sparse-checkout set --no-cone "/lib/$(basename $*)/libcronet$(suffix $*)"
	git -C $(BINDIR)/.cronet-go-lib fetch -q --depth=1 --filter=blob:none origin $(CRONET_GO_VERSION)
	git -C $(BINDIR)/.cronet-go-lib checkout -q FETCH_HEAD
	cp $(BINDIR)/.cronet-go-lib/lib/$(basename $*)/libcronet$(suffix $*) $(BINDIR)/
	rm -rf $(BINDIR)/.cronet-go-lib

cronet-%:
	$(MAKE) ARCH=$* build-cronet

build-cronet:
# 	rm -rf $(CRONET_DIR)
	git init $(CRONET_DIR) || echo "dir exist"
	cd $(CRONET_DIR) && \
	git remote add origin https://github.com/sagernet/cronet-go.git ||echo "remote exist"; \
	git fetch --depth=1 origin $(CRONET_GO_VERSION) && \
	git checkout FETCH_HEAD && \
	git submodule update --init --recursive --depth=1 && \
	if [ "$${VARIANT}" = "musl" ]; then \
		go run ./cmd/build-naive --target=linux/$(ARCH) --libc=musl download-toolchain && \
		go run ./cmd/build-naive --target=linux/$(ARCH) --libc=musl env > cronet.env; \
	else \
		go run ./cmd/build-naive --target=linux/$(ARCH) download-toolchain && \
		go run ./cmd/build-naive --target=linux/$(ARCH) env > cronet.env; \
	fi

################################
# Generic Linux Builder
################################
linux-%:
	$(MAKE) ARCH=$* build-linux

define load_cronet_env
set -a; \
while IFS= read -r line; do \
    key=$${line%%=*}; \
    value=$${line#*=}; \
    export "$$key=$$value"; \
	echo "$$key=$$value"; \
done < $(CRONET_DIR)/cronet.env; \
set +a;
endef

build-linux: prepare
	mkdir -p $(BINDIR)/lib

	$(load_cronet_env)
	FINAL_TAGS=$(TAGS); \
	if [ "$${VARIANT}" = "musl" ]; then \
		FINAL_TAGS=$${FINAL_TAGS},with_musl; \
	elif [ "$${VARIANT}" = "purego" ]; then \
		FINAL_TAGS="$${FINAL_TAGS},with_purego"; \
	fi; \
	echo "FinalTags: $$FINAL_TAGS"; \
	GOOS=linux GOARCH=$(ARCH) $(GOBUILDLIB) -tags $${FINAL_TAGS} -o $(BINDIR)/lib/$(LIBNAME).so ./platform/desktop ;\
	
	echo "Core library built, now building CLI with CGO linking to core library"
	mkdir lib
	cp $(BINDIR)/lib/$(LIBNAME).so ./lib/$(LIBNAME).so

	GOOS=linux GOARCH=$(ARCH) CGO_LDFLAGS="./lib/$(LIBNAME).so -Wl,-rpath,\$$ORIGIN/lib -fuse-ld=lld" $(GOBUILDSRV) -o $(BINDIR)/$(CLINAME) ./cmd/bydll
	
	rm -rf ./lib/*.so
	chmod +x $(BINDIR)/$(CLINAME)
	if [ ! -f $(BINDIR)/lib/$(LIBNAME).so -o ! -f $(BINDIR)/$(CLINAME) ]; then \
		echo "Error: $(LIBNAME).so or $(CLINAME) not built"; \
		ls -R $(BINDIR); \
		exit 1; \
	fi
# 	make webui


linux-custom: prepare  install_cronet
	mkdir -p $(BINDIR)/
	#env GOARCH=mips $(GOBUILDSRV) -o $(BINDIR)/$(CLINAME) ./cmd/
	$(load_cronet_env)
	go build -ldflags="$(LDFLAGS)" -trimpath -tags $(TAGS) -o $(BINDIR)/$(CLINAME) ./cmd/main
	chmod +x $(BINDIR)/$(CLINAME)
	make webui

macos-amd64:
	env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.11 -O2" CGO_LDFLAGS="-mmacosx-version-min=10.11 -O2 -lpthread" CGO_ENABLED=1 go build -trimpath -ldflags="$(EMBED_LDFLAGS)" -tags $(TAGS),$(MACOS_ADD_TAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-amd64.dylib ./platform/desktop
macos-arm64:
	env GOOS=darwin GOARCH=arm64 CGO_CFLAGS="-mmacosx-version-min=10.11 -O2" CGO_LDFLAGS="-mmacosx-version-min=10.11 -O2 -lpthread" CGO_ENABLED=1 go build -trimpath -ldflags="$(EMBED_LDFLAGS)" -tags $(TAGS),$(MACOS_ADD_TAGS) -buildmode=c-shared -o $(BINDIR)/$(LIBNAME)-arm64.dylib ./platform/desktop
	
macos: prepare macos-amd64 macos-arm64 
	
	lipo -create $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME)-arm64.dylib -output $(BINDIR)/$(LIBNAME).dylib
	cp $(BINDIR)/$(LIBNAME).dylib ./$(LIBNAME).dylib 
	mv $(BINDIR)/$(LIBNAME)-arm64.h $(BINDIR)/desktop.h 
	# env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.15" CGO_LDFLAGS="-mmacosx-version-min=10.15" CGO_LDFLAGS="bin/$(LIBNAME).dylib"  CGO_ENABLED=1 $(GOBUILDSRV)  -o $(BINDIR)/$(CLINAME) ./cmd/bydll
	# rm ./$(LIBNAME).dylib
	# chmod +x $(BINDIR)/$(CLINAME)

prepare: 
	go mod tidy

clean:
	rm $(BINDIR)/*




.PHONY: release
release: # Create a new tag for release.	
	@bash -c '.github/change_version.sh'
	


