@echo off
setlocal enableextensions

set SHARE=
set RESULT=
set OUT=
set EXITFILE=
set STATUS=FAIL
set EXITCODE=1

echo go-legacy-winxp XP smoke starting > C:\OEM\started.txt

REM Wait for Samba share from dockur host.
for /L %%i in (1,1,60) do (
  if exist \\host.lan\Data\ (
    set SHARE=\\host.lan\Data
    goto share_ready
  )
  net use Z: \\host.lan\Data /persistent:no >nul 2>&1
  if exist Z:\ (
    set SHARE=Z:
    goto share_ready
  )
  ping -n 6 127.0.0.1 >nul
)

echo share unavailable > C:\OEM\share-error.txt
goto run_tests

:share_ready
set RESULT=%SHARE%\result.txt
set OUT=%SHARE%\smoke.out
set EXITFILE=%SHARE%\smoke.exit
echo starting > "%RESULT%"

:run_tests
if not exist C:\OEM\xp-smoke-386.exe (
  echo missing C:\OEM\xp-smoke-386.exe > C:\OEM\smoke-386.out
  set EXITCODE=1
  goto write_result
)

C:\OEM\xp-smoke-386.exe > C:\OEM\smoke-386.out 2>&1
set EXITCODE=%ERRORLEVEL%
if %EXITCODE% neq 0 goto write_result
set STATUS=PASS

:write_result
if not "%SHARE%"=="" (
  echo %STATUS% > "%RESULT%"
  echo %EXITCODE% > "%EXITFILE%"
  if exist C:\OEM\smoke-386.out copy /Y C:\OEM\smoke-386.out "%OUT%" >nul
  if not exist C:\OEM\smoke-386.out echo no smoke output captured > "%OUT%"
)

REM Boot-time retest hook for cached disks.
if not exist "%USERPROFILE%\Start Menu\Programs\Startup" mkdir "%USERPROFILE%\Start Menu\Programs\Startup"
echo @echo off > "%USERPROFILE%\Start Menu\Programs\Startup\go-xp-retest.bat"
echo if exist \\host.lan\Data\run.bat call \\host.lan\Data\run.bat >> "%USERPROFILE%\Start Menu\Programs\Startup\go-xp-retest.bat"
echo if exist Z:\run.bat call Z:\run.bat >> "%USERPROFILE%\Start Menu\Programs\Startup\go-xp-retest.bat"

if not "%SHARE%"=="" (
  if "%STATUS%"=="PASS" shutdown -s -t 15
)

endlocal
