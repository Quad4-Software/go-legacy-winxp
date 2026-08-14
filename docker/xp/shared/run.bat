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

if exist "%SHARE%\xp-smoke-386.exe" (
  "%SHARE%\xp-smoke-386.exe" > "%SHARE%\smoke.out" 2>&1
  set EXITCODE=!ERRORLEVEL!
  echo !EXITCODE! > "%SHARE%\smoke.exit"
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

:done
endlocal
