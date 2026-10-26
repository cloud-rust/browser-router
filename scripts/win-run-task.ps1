# Run a script in the logged-on desktop session via a scheduled task; output goes to <script>.out.
param([string]$Script)
$out = "$HOME\$Script.out"; Remove-Item $out -ErrorAction SilentlyContinue
$a = New-ScheduledTaskAction -Execute "cmd.exe" -Argument "/c powershell -NoProfile -ExecutionPolicy Bypass -File $HOME\$Script > $out 2>&1"
$p = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive
Register-ScheduledTask -TaskName "browser-router-$Script" -Action $a -Principal $p -Force | Out-Null
Start-ScheduledTask -TaskName "browser-router-$Script"
