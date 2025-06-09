# Brummer Quick Installation Script for Windows
# Downloads and installs the latest Brummer binary

$ErrorActionPreference = 'Stop'

# Configuration
$repo = "standardbeagle/brummer"
$binaryName = "brum"
$installDir = "$env:USERPROFILE\.local\bin"
$version = "latest"

# Colors for output
function Write-Info { Write-Host "[INFO]" -ForegroundColor Blue -NoNewline; Write-Host " $args" }
function Write-Success { Write-Host "[SUCCESS]" -ForegroundColor Green -NoNewline; Write-Host " $args" }
function Write-Warning { Write-Host "[WARNING]" -ForegroundColor Yellow -NoNewline; Write-Host " $args" }
function Write-Error { Write-Host "[ERROR]" -ForegroundColor Red -NoNewline; Write-Host " $args" }

Write-Host "`n🐝 Brummer Quick Installer" -ForegroundColor Yellow
Write-Host "==========================" -ForegroundColor Yellow
Write-Host ""

# Check if already installed
if (Get-Command $binaryName -ErrorAction SilentlyContinue) {
    $currentVersion = & $binaryName --version 2>$null
    Write-Warning "Brummer is already installed (version: $currentVersion)"
    $response = Read-Host "Do you want to reinstall/update? (y/N)"
    if ($response -ne 'y' -and $response -ne 'Y') {
        Write-Info "Installation cancelled"
        exit 0
    }
}

# Get latest release version
Write-Info "Fetching latest release information..."
try {
    if ($version -eq "latest") {
        $releaseUrl = "https://api.github.com/repos/$repo/releases/latest"
        $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{"User-Agent"="brummer-installer"}
        $version = $release.tag_name
    }
    Write-Info "Installing version: $version"
} catch {
    Write-Error "Failed to fetch latest release: $_"
    exit 1
}

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
Write-Info "Detected architecture: $arch"

# Construct download URL
$downloadUrl = "https://github.com/$repo/releases/download/$version/$binaryName-windows-$arch.exe"
Write-Info "Download URL: $downloadUrl"

# Create install directory
if (!(Test-Path $installDir)) {
    Write-Info "Creating install directory: $installDir"
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Download binary
Write-Info "Downloading Brummer..."
$tempFile = Join-Path $env:TEMP "$binaryName.exe"
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
    Write-Success "Download complete"
} catch {
    Write-Error "Failed to download binary: $_"
    exit 1
}

# Install binary
Write-Info "Installing to $installDir..."
$targetPath = Join-Path $installDir "$binaryName.exe"
Move-Item -Path $tempFile -Destination $targetPath -Force
Write-Success "Installation complete!"

# Check if install directory is in PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Warning "$installDir is not in your PATH"
    Write-Info "Adding to PATH..."
    
    $newPath = if ($userPath) { "$userPath;$installDir" } else { $installDir }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + $newPath
    
    Write-Success "Added to PATH. Restart your terminal for changes to take effect."
}

# Verify installation
if (Get-Command $binaryName -ErrorAction SilentlyContinue) {
    Write-Success "Brummer is ready to use!"
    Write-Info "Run 'brum' to get started"
} else {
    Write-Info "Run '$targetPath' to get started"
    Write-Info "Restart your terminal to use 'brum' command directly"
}

Write-Host "`n"
Write-Success "Installation complete! 🎉"
Write-Host "`nNext steps:"
Write-Host "1. Run 'brum' in a project directory with package.json"
Write-Host "2. Press '?' in the TUI for help"
Write-Host "3. Visit https://github.com/$repo for documentation"