@echo off
REM an example command runner for windows.
REM run the following once, to prevent popup errors
REM reg add "HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Error Reporting" /v "DontShowUI" /t REG_DWORD /d 1 /f
REM set PATH=%~dp0;%path%

REM assumes 1 installation per worker for isolation
set app_path=%~dp0\..\app\install_dir_%JOBRUNNER_WORKER_ID%
cd %app_path%
echo copying %JOBRUNNER_REQUEST_DATA_FN% to app_in.data
copy /Y %JOBRUNNER_REQUEST_DATA_FN% app_in.data 1>nul
del /q app_out*.data 2>nul

echo Running app
app.exe

echo waiting for app_out.data to appear
waitforfile.exe 10 . app_out.data
echo will copy result to %JOBRUNNER_RESPONSE_DATA_FN%
copy /y app_out.data %JOBRUNNER_RESPONSE_DATA_FN% 1>nul
echo done
