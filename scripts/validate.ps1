[CmdletBinding()]
param([string]$GoExecutable='go',[switch]$Race,[switch]$Scan)
$ErrorActionPreference='Stop'
$project=Join-Path $(if($env:DEVELOPMENT_ROOT){$env:DEVELOPMENT_ROOT}elseif($env:RUNNER_TEMP){Join-Path $env:RUNNER_TEMP 'development'}else{'V:\Development'}) 'go-document-docx'
$run=[DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss')
$output=Join-Path $project "artifacts\runs\validate-$run"
$scratch=Join-Path $project "tmp\validate-$run"
$names=@('TEMP','TMP','TMPDIR','GOTMPDIR','GOCACHE','GOMODCACHE','GOWORK','GOTOOLCHAIN','GOFLAGS','CGO_ENABLED')
$saved=@{};foreach($key in $names){$saved[$key]=[Environment]::GetEnvironmentVariable($key,'Process')}
function Invoke-Go([string[]]$Arguments){& $GoExecutable @Arguments;if($LASTEXITCODE -ne 0){throw "Go check failed: $($Arguments -join ' ')"}}
Push-Location (Split-Path -Parent $PSScriptRoot)
try{
 $env:TEMP=$scratch;$env:TMP=$scratch;$env:TMPDIR=$scratch;$env:GOTMPDIR=Join-Path $scratch 'go-build'
 $env:GOCACHE=Join-Path $project 'caches\go-build';$env:GOMODCACHE=Join-Path $project 'caches\go-modules'
 New-Item -ItemType Directory -Path $scratch,$output,$env:GOTMPDIR,$env:GOCACHE,$env:GOMODCACHE -Force|Out-Null
 $env:GOWORK='off';$env:GOTOOLCHAIN='local';$env:GOFLAGS='-mod=readonly';$env:CGO_ENABLED='0'
 if((& $GoExecutable env GOVERSION) -ne 'go1.27.3'){throw 'Go 1.27.3 is required'}
 $goroot=& $GoExecutable env GOROOT
 $unformatted=@(& (Join-Path $goroot 'bin\gofmt.exe') -l .)
 if($LASTEXITCODE -ne 0 -or $unformatted.Count){throw "Formatting required: $unformatted"}
 Invoke-Go -Arguments @('test','-count=1','./...');Invoke-Go -Arguments @('vet','./...');Invoke-Go -Arguments @('mod','verify');Invoke-Go -Arguments @('build','./...')
 if($Race){$env:CGO_ENABLED='1';try{Invoke-Go -Arguments @('test','-race','-count=1','./...')}finally{$env:CGO_ENABLED='0'}}
 Invoke-Go -Arguments @('build','-trimpath','-buildvcs=false','-o',(Join-Path $output 'report-example.exe'),'./examples/report')
 & $GoExecutable version -m (Join-Path $output 'report-example.exe') | Set-Content -LiteralPath (Join-Path $output 'build-metadata.txt')
 & $GoExecutable env -json GOVERSION GOOS GOARCH CGO_ENABLED GOWORK GOFLAGS | Set-Content -LiteralPath (Join-Path $output 'build-settings.json')
 & $GoExecutable list -m -json all | Set-Content -LiteralPath (Join-Path $output 'modules.json')
 if($Scan){Invoke-Go -Arguments @('run','golang.org/x/vuln/cmd/govulncheck@v1.7.0','./...');Invoke-Go -Arguments @('run','golang.org/x/vuln/cmd/govulncheck@v1.7.0','-mode=binary',(Join-Path $output 'report-example.exe'))}
 Write-Output "Validated source and native build. Evidence: $output"
}finally{foreach($key in $names){[Environment]::SetEnvironmentVariable($key,$saved[$key],'Process')};Pop-Location}
