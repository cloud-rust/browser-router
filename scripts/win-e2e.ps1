# Windows end-to-end check, run in the desktop session. Mode "progid" calls the registered command
# line directly; mode "shell" opens links like any app would, via the default browser.
param([string]$Mode = "shell")
$T = [DateTimeOffset]::Now.ToUnixTimeSeconds()
$srv = "$env:TEMP\probe-srv.ps1"
@'
$l = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 3000); $l.Start()
while ($true) { $c = $l.AcceptTcpClient(); $s = $c.GetStream(); $buf = New-Object byte[] 4096; [void]$s.Read($buf, 0, 4096)
  $b = [Text.Encoding]::ASCII.GetBytes("HTTP/1.1 200 OK`r`nContent-Length: 2`r`nConnection: close`r`n`r`nok"); $s.Write($b, 0, $b.Length); $c.Close() }
'@ | Set-Content $srv
$server = Start-Process powershell -ArgumentList "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $srv -WindowStyle Hidden -PassThru
Start-Sleep 2
$cmdline = (Get-ItemProperty "HKCU:\Software\Classes\BrowserRouterPoC.URL\shell\open\command").'(default)'
$exe = ($cmdline -split '"')[1]
"mode: $Mode"
"https default: " + (Get-ItemProperty "HKCU:\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice" -ErrorAction SilentlyContinue).ProgId
foreach ($u in "http://localhost:3000/?probe=$T-dev", "https://example.com/acme/?probe=$T-acme", "https://example.org/?probe=$T-other") {
  if ($Mode -eq "progid") { Start-Process $exe -ArgumentList "`"$u`"" } else { Start-Process $u }
  Start-Sleep 10
}
Start-Sleep 20
Get-Process firefox, vivaldi -ErrorAction SilentlyContinue | ForEach-Object { [void]$_.CloseMainWindow() }; Start-Sleep 8
Get-Process firefox, vivaldi -ErrorAction SilentlyContinue | Stop-Process -Force
Stop-Process -Id $server.Id -Force
"--- router.log"; Get-Content "$env:APPDATA\browser-router-poc\router.log" | Select-String "probe=$T"
"--- landed"
function Probes($file, $label) {
  $found = foreach ($f in @($file, "$file-wal")) { if (Test-Path $f) {
    $text = [Text.Encoding]::ASCII.GetString([IO.File]::ReadAllBytes($f))
    [regex]::Matches($text, "probe=$T-(dev|acme|other)") | ForEach-Object { $_.Groups[1].Value } } }
  $found | Sort-Object -Unique | ForEach-Object { "  [$label]: $_" }
}
Probes "$env:TEMP\ffp\places.sqlite" "firefox"
Get-ChildItem "$env:TEMP\vud" -Directory | ForEach-Object { Probes (Join-Path $_.FullName "History") "vivaldi $($_.Name)" }
