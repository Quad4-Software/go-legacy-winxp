@echo off
setlocal enableextensions enabledelayedexpansion

set SHARE=Z:
if not exist "%SHARE%\reticulum-go-winxp.exe" (
  if exist \\host.lan\Data\reticulum-go-winxp.exe set SHARE=\\host.lan\Data
)
if not exist "%SHARE%\config" (
  echo missing config > "%SHARE%\live-result.txt"
  echo missing Z:\config
  goto :eof
)
if not exist "%SHARE%\reticulum-go-winxp.exe" (
  echo missing reticulum-go > "%SHARE%\live-result.txt"
  echo missing reticulum-go-winxp.exe
  goto :eof
)

echo starting > "%SHARE%\live-result.txt"

if not exist C:\rns mkdir C:\rns
copy /Y "%SHARE%\reticulum-go-winxp.exe" C:\rns\reticulum-go.exe >nul
copy /Y "%SHARE%\config" C:\rns\config >nul

taskkill /F /IM reticulum-go.exe >nul 2>&1

cd /d C:\rns
start "reticulum-go" reticulum-go.exe --config C:\rns --debug 4

echo reticulum-go daemon started with MeshChatX TCP hubs
echo waiting for shared instance and outbound TCP
ping -n 36 127.0.0.1 >nul

C:\rns\reticulum-go.exe --version > "%SHARE%\live-version.txt" 2>&1
C:\rns\reticulum-go.exe status -config C:\rns -a -t > "%SHARE%\live-status.txt" 2>&1
echo !ERRORLEVEL! > "%SHARE%\live-status.exit"
C:\rns\reticulum-go.exe status -config C:\rns -a -json > "%SHARE%\live-status.json" 2>&1

if exist C:\rns\logfile\reticulum.log copy /Y C:\rns\logfile\reticulum.log "%SHARE%\live-daemon.log" >nul

cls
echo ========================================
echo reticulum-go live test on Windows XP
echo config C:\rns MeshChatX TCP / backbone
echo ========================================
echo.
echo === version ===
type "%SHARE%\live-version.txt"
echo.
echo === status ===
type "%SHARE%\live-status.txt"
echo.

findstr /C:"Status" "%SHARE%\live-status.txt" >nul
if errorlevel 1 (
  echo FAIL > "%SHARE%\live-result.txt"
  echo status RPC failed
) else (
  echo DAEMON > "%SHARE%\live-result.txt"
  echo daemon RPC answered, host script checks hub Up
)

echo live-result:
type "%SHARE%\live-result.txt"
echo.
echo window left open for screenshot
ping -n 301 127.0.0.1 >nul
endlocal
