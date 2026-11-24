@echo off
setlocal enableextensions enabledelayedexpansion

set SHARE=\\host.lan\Data
set STATUS=FAIL
set EXITCODE=1
if not exist "%SHARE%" (
  if exist Z:\ set SHARE=Z:
)
if not exist "%SHARE%" goto done

echo starting > "%SHARE%\result.txt"

REM Launch the headless agent so scripts/xp-agent.sh can drive the guest.
if exist "%SHARE%\xp-agent.exe" (
  start "xp-agent" /min "%SHARE%\xp-agent.exe" "%SHARE%"
)

if exist "%SHARE%\reticulum-go-winxp.exe" (
  "%SHARE%\reticulum-go-winxp.exe" --version > "%SHARE%\reticulum.out" 2>&1
  set EXITCODE=!ERRORLEVEL!
  if !EXITCODE! neq 0 (
    echo reticulum-go --version failed > "%SHARE%\smoke.out"
    type "%SHARE%\reticulum.out" >> "%SHARE%\smoke.out"
    echo !EXITCODE! > "%SHARE%\smoke.exit"
    goto write_fail
  )
  findstr /C:"reticulum-go" "%SHARE%\reticulum.out" >nul
  if errorlevel 1 (
    echo reticulum-go version output missing marker > "%SHARE%\smoke.out"
    type "%SHARE%\reticulum.out" >> "%SHARE%\smoke.out"
    set EXITCODE=1
    echo 1 > "%SHARE%\smoke.exit"
    goto write_fail
  )
)

if exist "%SHARE%\xp-smoke-386.exe" (
  "%SHARE%\xp-smoke-386.exe" > "%SHARE%\smoke.tmp" 2>&1
  set EXITCODE=!ERRORLEVEL!
  echo !EXITCODE! > "%SHARE%\smoke.exit"
  if exist "%SHARE%\reticulum.out" (
    echo reticulum-go --version ok > "%SHARE%\smoke.out"
    type "%SHARE%\reticulum.out" >> "%SHARE%\smoke.out"
  ) else (
    echo. > "%SHARE%\smoke.out"
  )
  type "%SHARE%\smoke.tmp" >> "%SHARE%\smoke.out"
  if !EXITCODE! neq 0 goto write_fail
  set STATUS=PASS
  echo PASS > "%SHARE%\result.txt"
  goto write_log
) else (
  echo missing xp-smoke-386.exe > "%SHARE%\smoke.out"
  echo 1 > "%SHARE%\smoke.exit"
  goto write_fail
)

:write_fail
echo FAIL > "%SHARE%\result.txt"
set STATUS=FAIL

:write_log
echo status=%STATUS% > "%SHARE%\smoke.log"
echo exit=%EXITCODE% >> "%SHARE%\smoke.log"
echo --- stdout stderr --- >> "%SHARE%\smoke.log"
if exist "%SHARE%\smoke.out" type "%SHARE%\smoke.out" >> "%SHARE%\smoke.log"

REM Toolchain smoke: only when a staged GOROOT is present. Writes its own
REM result files so failures do not mask the smoke result above.
if exist "%SHARE%\go\bin\go.exe" (
  set GOROOT=%SHARE%\go
  set PATH=%SHARE%\go\bin;%PATH%
  "%SHARE%\go\bin\go.exe" version > "%SHARE%\go-version.out" 2>&1
  set GOEXIT=!ERRORLEVEL!
  "%SHARE%\go\bin\go.exe" env GOOS GOARCH GOROOT >> "%SHARE%\go-version.out" 2>&1
  if exist "%SHARE%\app\hello.go" (
    "%SHARE%\go\bin\go.exe" build -o "%TEMP%\xp-hello.exe" "%SHARE%\app\hello.go" > "%SHARE%\go-build.out" 2>&1
    if !ERRORLEVEL! equ 0 (
      "%TEMP%\xp-hello.exe" > "%SHARE%\go-run.out" 2>&1
      set RUNEXIT=!ERRORLEVEL!
    ) else (
      set RUNEXIT=build-failed
    )
  ) else (
    set RUNEXIT=no-app
  )
  (
    echo go-version exit=%GOEXIT%
    type "%SHARE%\go-version.out"
    echo go-build/run exit=%RUNEXIT%
    if exist "%SHARE%\go-build.out" type "%SHARE%\go-build.out"
    if exist "%SHARE%\go-run.out" type "%SHARE%\go-run.out"
  ) > "%SHARE%\go-smoke.out" 2>&1
  echo done > "%SHARE%\go-smoke.done"
)

:done
endlocal
