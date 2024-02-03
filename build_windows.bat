@echo off
set GOOS=windows
set GOARCH=amd64
set CC=x86_64-w64-mingw32-gcc
set CGO_ENABLED=1
curl http://localhost:18020/exit || echo "Exited"
del bin\libcore.dll bin\HiddifyService.exe
set CGO_LDFLAGS=
go build -trimpath -tags with_gvisor,with_quic,with_wireguard,with_ech,with_utls,with_clash_api,with_grpc -ldflags="-w -s" -buildmode=c-shared -o bin/libcore.dll ./custom
go get github.com/akavel/rsrc
go install github.com/akavel/rsrc

rsrc -manifest admin_service\cmd\admin_service.manifest -ico ..\assets\images\tray_icon_connected.ico -o admin_service\cmd\admin_service.syso

copy bin\libcore.dll .
set CGO_LDFLAGS="libcore.dll"
go build  -o bin/HiddifyService.exe ./admin_service/cmd/
del libcore.dll
