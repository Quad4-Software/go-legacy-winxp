@echo off
setlocal enableextensions

set SHARE=\\host.lan\Data
if not exist "%SHARE%" (
  if exist Z:\ set SHARE=Z:
)
if not exist "%SHARE%" goto done

echo starting > "%SHARE%\result.txt"

if exist "%SHARE%\xp-smoke-386.exe" (
  "%SHARE%\xp-smoke-386.exe" > "%SHARE%\smoke.out" 2>&1
  if errorlevel 1 (
    echo FAIL > "%SHARE%\result.txt"
    goto done
  )
  echo PASS > "%SHARE%\result.txt"
) else (
  echo missing xp-smoke-386.exe > "%SHARE%\smoke.out"
  echo FAIL > "%SHARE%\result.txt"
)

:done
endlocal
