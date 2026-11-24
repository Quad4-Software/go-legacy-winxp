@echo off
set GOROOT=\\host.lan\Data\go
set PATH=%GOROOT%\bin;%PATH%
set CGO_ENABLED=0
set GOCACHE=C:\gocache
%GOROOT%\bin\go.exe run \\host.lan\Data\app\gorun-app.go
