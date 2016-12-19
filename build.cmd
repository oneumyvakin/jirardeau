set PKGNAME=github.com/oneumyvakin/jirardeau
set LOCALPATH=%~dp0

goimports.exe -w .
go fmt %PKGNAME%
rem install staticcheck from go get -u honnef.co/go/staticcheck/cmd/staticcheck
staticcheck.exe %PKGNAME%
go vet %PKGNAME%

set GOOS=linux
set GOARCH=amd64
go build -o jirardeau.%GOARCH% %PKGNAME%

set GOOS=windows
set GOARCH=amd64
go build -o jirardeau.exe %PKGNAME%