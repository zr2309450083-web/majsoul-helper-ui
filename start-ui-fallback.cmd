@echo off
cd /d "%~dp0"
start "" /min "mahjong-helper-ui.exe" -majsoul -p 12121
timeout /t 2 >nul
start "" "http://127.0.0.1:12122/ui"
