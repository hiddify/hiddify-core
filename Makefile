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
TAGS=with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_grpc,with_awg,tfogo_checklinkname0,with_naive_outbound,with_conntrack,with_embedded_tor,with_openvpn,with_openconnect,with_masque,with_kcp,with_sudoku,with_bond,with_call
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
# Per-build UUID so the app can match its downloaded core to the tunnel service (Go only; Dart never embeds it).
ifeq ($(strip $(HIDDIFY_BUILD_ID)),)
HIDDIFY_BUILD_ID := $(shell (uuidgen 2>/dev/null || openssl rand -hex 16) | tr '[:upper:]' '[:lower:]' | tr -d '\n')
endif
EMBED_LDFLAGS+=-X github.com/hiddify/hiddify-core/v2/hcore/buildid.ID=$(strip $(HIDDIFY_BUILD_ID))
LDFLAGS=-w -s -checklinkname=0 -buildid= $${CODE_VERSION} $(EMBED_LDFLAGS)
GOBUILDLIB=CGO_ENABLED=1 go build -trimpath -ldflags="$(LDFLAGS)" -buildmode=c-shared
GOBUILDSRV=CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -trimpath -tags $(TAGS)

# Windows PE files: keep the Go symbol table (no -s), drop only DWARF (-w).
# Symbol-less, fully stripped Go PE files are a classic feature of ML/heuristic AV
# verdicts (Webroot W32.Trojan.Gen, Bkav W32.Malware.*, DeepInstinct) on VirusTotal.
# The size cost is a few percent; the app logic is unchanged.
WINDOWS_LDFLAGS=-w -checklinkname=0 -buildid= $${CODE_VERSION} $(EMBED_LDFLAGS)
GOBUILDLIB_WINDOWS=CGO_ENABLED=1 go build -trimpath -ldflags="$(WINDOWS_LDFLAGS)" -buildmode=c-shared
GOBUILDSRV_WINDOWS=CGO_ENABLED=1 go build -ldflags="$(WINDOWS_LDFLAGS)" -trimpath -tags $(TAGS)

# go-winres embeds VERSIONINFO (CompanyName/ProductName/...), the icon and an
# application manifest into the Windows PE files. PE files without a version
# resource are another strong "unknown dropper" signal for AV engines.
GO_WINRES_VERSION=v0.3.3
# "v3.2.1-5-gabcdef" -> "3.2.1.5"; anything unparsable -> "0.0.0.0"
WIN_FILE_VERSION=$(shell printf '%s\n' "$(VERSION)" | sed -E 's/^v//; s/-([0-9]+)-g[0-9a-f]+.*$$/.\1/; s/[^0-9.].*$$//; s/\.+$$//' | awk -F. '{ printf "%d.%d.%d.%d", $$1, $$2, $$3, $$4 }')

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
# Generates rsrc_windows_amd64.syso next to the Windows main packages. Go links
# .syso files automatically (only for GOOS=windows GOARCH=amd64 thanks to the suffix).
.PHONY: windows-winres
windows-winres:
	go install github.com/tc-hib/go-winres@$(GO_WINRES_VERSION)
	rm -f ./platform/desktop/rsrc_windows_*.syso ./cmd/clilauncher/rsrc_windows_*.syso ./cmd/adminlauncher/rsrc_windows_*.syso ./cmd/bydll/rsrc_windows_*.syso ./cmd/bydll/cli.syso
	echo "Windows PE file version: $(WIN_FILE_VERSION) (from $(VERSION))"
	$$(go env GOPATH)/bin/go-winres make --in ./release/winres/core.json --out ./platform/desktop/rsrc --arch amd64 --file-version "$(WIN_FILE_VERSION)" --product-version "$(WIN_FILE_VERSION)"
	$$(go env GOPATH)/bin/go-winres make --in ./release/winres/cli.json --out ./cmd/clilauncher/rsrc --arch amd64 --file-version "$(WIN_FILE_VERSION)" --product-version "$(WIN_FILE_VERSION)"
	$$(go env GOPATH)/bin/go-winres make --in ./release/winres/admin.json --out ./cmd/adminlauncher/rsrc --arch amd64 --file-version "$(WIN_FILE_VERSION)" --product-version "$(WIN_FILE_VERSION)"
	ls -la ./platform/desktop/rsrc_windows_amd64.syso ./cmd/clilauncher/rsrc_windows_amd64.syso ./cmd/adminlauncher/rsrc_windows_amd64.syso

windows-amd64: prepare windows-winres
	rm -rf $(BINDIR)/*
	$(MAKE) BINDIR=$(BINDIR) fetch-libcronet-windows_amd64.dll
	env GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc  $(GOBUILDLIB_WINDOWS) -tags $(TAGS),$(WINDOWS_ADD_TAGS)   -o $(BINDIR)/$(LIBNAME).dll ./platform/desktop
	echo "core built, now building cli" 
	ls -R $(BINDIR)/
	# HiddifyCli.exe is a small C launcher that calls parseCli() in the DLL (see
	# cmd/clilauncher/main.c). The former Go stub (cmd/bydll, still used on Linux)
	# was flagged by ML antivirus engines whatever its link flags; the C launcher
	# scans clean. The go-winres .syso (icon, VERSIONINFO, manifest) is linked in.
	x86_64-w64-mingw32-gcc -O2 -Wall -mconsole -o $(BINDIR)/$(CLINAME).exe ./cmd/clilauncher/main.c ./cmd/clilauncher/rsrc_windows_amd64.syso -L $(BINDIR) -l:$(LIBNAME).dll -lshell32
	# HiddifyAdmin.exe: GUI-subsystem launcher with a requireAdministrator manifest and
	# the app icon; the only binary that triggers a UAC prompt (VPN service install).
	x86_64-w64-mingw32-gcc -O2 -Wall -mwindows -municode -o $(BINDIR)/HiddifyAdmin.exe ./cmd/adminlauncher/main.c ./cmd/adminlauncher/rsrc_windows_amd64.syso -L $(BINDIR) -l:$(LIBNAME).dll -lshell32
	if [ ! -f $(BINDIR)/$(LIBNAME).dll -o ! -f $(BINDIR)/$(CLINAME).exe -o ! -f $(BINDIR)/HiddifyAdmin.exe -o ! -f $(BINDIR)/libcronet.dll ]; then \
		echo "Error: $(LIBNAME).dll, $(CLINAME).exe, HiddifyAdmin.exe or libcronet.dll missing"; \
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

# Packet-tunnel / system-extension bind (same tags as hiddify-app macos/scripts/build-admin-bgservice-core.sh).
TUNNEL_TAGS=with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_grpc,with_awg,tfogo_checklinkname0,with_naive_outbound,with_conntrack,with_dhcp

.PHONY: macos-tunnel
macos-tunnel:
	mkdir -p $(BINDIR)
	go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.1.12
	go install -v github.com/sagernet/gomobile/cmd/gobind@v0.1.12
	gomobile bind -v -target macos -macosversion 12.0 -libname HiddifyTunnelCore -tags $(TUNNEL_TAGS) \
		-trimpath -ldflags "$(LDFLAGS)" \
		-o $(BINDIR)/HiddifyTunnelCore.xcframework \
		github.com/sagernet/sing-box/experimental/libbox github.com/hiddify/hiddify-core/v2/hcore/tunnel

macos: prepare macos-amd64 macos-arm64 macos-tunnel
	
	lipo -create $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME)-arm64.dylib -output $(BINDIR)/$(LIBNAME).dylib
	cp $(BINDIR)/$(LIBNAME).dylib ./$(LIBNAME).dylib 
	mv $(BINDIR)/$(LIBNAME)-arm64.h $(BINDIR)/desktop.h 
	test -d $(BINDIR)/HiddifyTunnelCore.xcframework
	# env GOOS=darwin GOARCH=amd64 CGO_CFLAGS="-mmacosx-version-min=10.15" CGO_LDFLAGS="-mmacosx-version-min=10.15" CGO_LDFLAGS="bin/$(LIBNAME).dylib"  CGO_ENABLED=1 $(GOBUILDSRV)  -o $(BINDIR)/$(CLINAME) ./cmd/bydll
	# rm ./$(LIBNAME).dylib
	# chmod +x $(BINDIR)/$(CLINAME)

# Current host arch only; skips the tunnel/admin-bg xcframework (faster local Flutter runs).
.PHONY: macos-lite
macos-lite: prepare
	mkdir -p $(BINDIR)
	@arch=$$(uname -m); \
	if [ "$$arch" = "arm64" ]; then \
		$(MAKE) macos-arm64; \
		cp $(BINDIR)/$(LIBNAME)-arm64.dylib $(BINDIR)/$(LIBNAME).dylib; \
		mv $(BINDIR)/$(LIBNAME)-arm64.h $(BINDIR)/desktop.h; \
	elif [ "$$arch" = "x86_64" ]; then \
		$(MAKE) macos-amd64; \
		cp $(BINDIR)/$(LIBNAME)-amd64.dylib $(BINDIR)/$(LIBNAME).dylib; \
		mv $(BINDIR)/$(LIBNAME)-amd64.h $(BINDIR)/desktop.h; \
	else \
		echo "unsupported macOS arch: $$arch" >&2; exit 1; \
	fi
	cp $(BINDIR)/$(LIBNAME).dylib ./$(LIBNAME).dylib


prepare: 
	go mod tidy

clean:
	rm $(BINDIR)/*




.PHONY: release
release: # Create a new tag for release.	
	@bash -c '.github/change_version.sh'
	


