Option Explicit
Dim fso, sh, base, edge
Set fso = CreateObject("Scripting.FileSystemObject")
Set sh = CreateObject("WScript.Shell")
base = fso.GetParentFolderName(WScript.ScriptFullName)
sh.CurrentDirectory = base

' Start the helper hidden. It keeps the original HTTPS ingestion on 12121.
sh.Run """" & base & "\mahjong-helper-ui.exe"" -majsoul -p 12121", 0, False
WScript.Sleep 1500

edge = sh.ExpandEnvironmentStrings("%ProgramFiles(x86)%") & "\Microsoft\Edge\Application\msedge.exe"
If Not fso.FileExists(edge) Then
    edge = sh.ExpandEnvironmentStrings("%ProgramFiles%") & "\Microsoft\Edge\Application\msedge.exe"
End If

If fso.FileExists(edge) Then
    sh.Run """" & edge & """ --app=http://127.0.0.1:12122/ui --window-size=520,760 --window-position=1390,30", 1, False
Else
    sh.Run "cmd /c start """" http://127.0.0.1:12122/ui", 1, False
End If
