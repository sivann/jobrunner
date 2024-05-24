REM starts jobrunner.exe

rem script path= %~dp0
set PATH=%~dp0;%path%

set JOBRUNNER_CMD=%~dp0\runner.bat
set JOBRUNNER_NUM_WORKERS=2
set JOBRUNNER_HTTP_LISTEN_ADDRESS=0.0.0.0

jobrunner.exe 

