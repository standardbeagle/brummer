$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'brummer'
$url64 = 'https://github.com/beagle/brummer/releases/download/v0.1.0/brum-windows-amd64.exe'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  fileType       = 'exe'
  url64bit       = $url64
  checksum64     = 'PLACEHOLDER_SHA256_WINDOWS_AMD64'
  checksumType64 = 'sha256'
}

# Download and extract the binary
$binPath = Join-Path $toolsDir 'brum.exe'
Get-ChocolateyWebFile @packageArgs -FileFullPath $binPath

# Create shim
Install-BinFile -Name 'brum' -Path $binPath
Install-BinFile -Name 'brummer' -Path $binPath