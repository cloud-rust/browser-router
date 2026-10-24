# Windows test setup: install the router for the current user, write a test config whose Vivaldi
# rule uses an isolated data dir with a profile named "Acme", then register the router.
$dir = "$HOME\router-poc"; New-Item -ItemType Directory -Force $dir | Out-Null
Copy-Item "$HOME\router.exe" "$dir\router.exe" -Force
$udd = "$env:TEMP\vud"; $ffp = "$env:TEMP\ffp"
New-Item -ItemType Directory -Force $ffp | Out-Null

$ls = "$udd\Local State"
$state = Get-Content $ls -Raw | ConvertFrom-Json
$state.profile.info_cache."Profile 1".name = "Acme"
[IO.File]::WriteAllText($ls, ($state | ConvertTo-Json -Depth 100 -Compress))  # Vivaldi's own file: no BOM

$cfg = [ordered]@{
  rules = @(
    [ordered]@{ name = "development"; match = @('^https?://(localhost|127\.0\.0\.1|\[::1\]|[^/]+\.localhost)(:\d+)?([/?#]|$)')
                target = [ordered]@{ browser = "firefox"; args = @("-profile", $ffp) } },
    [ordered]@{ name = "acme"; match = @("acme")
                target = [ordered]@{ browser = "vivaldi"; profile = "Acme"; user_data_dir = $udd; args = @("--no-first-run") } }
  )
  fallback = [ordered]@{ browser = "firefox"; args = @("-profile", $ffp) }
}
$cdir = "$env:APPDATA\browser-router-poc"; New-Item -ItemType Directory -Force $cdir | Out-Null
# Written with a BOM on purpose: users editing with Windows tools produce the same.
$cfg | ConvertTo-Json -Depth 10 | Set-Content "$cdir\config.json" -Encoding UTF8
Remove-Item "$cdir\router.log" -ErrorAction SilentlyContinue
& "$dir\router.exe" register
"register exit: $LASTEXITCODE"
