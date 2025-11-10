# Cross-platform build script for Scanner Service (PowerShell)
# Builds binaries for Windows, Linux, and macOS

param(
    [switch]$SkipLinux,
    [switch]$SkipMacOS,
    [switch]$WindowsOnly
)

Write-Host "========================================" -ForegroundColor Green
Write-Host "Scanner Service - Cross-Platform Build" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

# Configuration
$AppName = "scanserver"
$Version = "1.0.0"
$BuildDir = "build"
$CmdPath = "cmd/scanserver"

# Clean build directory
Write-Host "Cleaning build directory..." -ForegroundColor Yellow
if (Test-Path $BuildDir) {
    Remove-Item -Path $BuildDir -Recurse -Force
}
New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null

# Build info
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$GitCommit = try { git rev-parse --short HEAD 2>$null } catch { "unknown" }

# LD FLAGS
$LDFlags = "-s -w"
$LDFlags += " -X main.Version=$Version"
$LDFlags += " -X main.BuildTime=$BuildTime"
$LDFlags += " -X main.GitCommit=$GitCommit"

# Build function
function Build-Platform {
    param(
        [string]$OS,
        [string]$Arch
    )

    $OutputName = $AppName
    if ($OS -eq "windows") {
        $OutputName = "${AppName}.exe"
    }

    $OutputPath = "${BuildDir}\${AppName}-${OS}-${Arch}\${OutputName}"
    $OutputDir = Split-Path -Parent $OutputPath

    Write-Host "Building for ${OS}/${Arch}..." -ForegroundColor Yellow

    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

    $env:CGO_ENABLED = "0"
    $env:GOOS = $OS
    $env:GOARCH = $Arch

    $BuildCmd = "go build -ldflags `"$LDFlags`" -o `"$OutputPath`" `".\${CmdPath}`""

    try {
        Invoke-Expression $BuildCmd

        if ($LASTEXITCODE -eq 0) {
            $Size = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✓ Built ${OS}/${Arch} successfully ($([math]::Round($Size, 2)) MB)" -ForegroundColor Green

            # Copy additional files
            if (Test-Path "web") {
                Copy-Item -Path "web" -Destination "${BuildDir}\${AppName}-${OS}-${Arch}\" -Recurse -Force
            }
            if (Test-Path "config.example.yaml") {
                Copy-Item -Path "config.example.yaml" -Destination "${BuildDir}\${AppName}-${OS}-${Arch}\" -Force
            }
            if (Test-Path "README.md") {
                Copy-Item -Path "README.md" -Destination "${BuildDir}\${AppName}-${OS}-${Arch}\" -Force
            }

            # Create archive
            Push-Location $BuildDir
            $ArchiveName = if ($OS -eq "windows") {
                "${AppName}-${OS}-${Arch}-v${Version}.zip"
            } else {
                "${AppName}-${OS}-${Arch}-v${Version}.tar.gz"
            }

            if ($OS -eq "windows") {
                Compress-Archive -Path "${AppName}-${OS}-${Arch}" -DestinationPath $ArchiveName -Force
                Write-Host "✓ Created $ArchiveName" -ForegroundColor Green
            } else {
                # For tar.gz, use tar if available on Windows (Git Bash, WSL)
                if (Get-Command tar -ErrorAction SilentlyContinue) {
                    tar -czf $ArchiveName "${AppName}-${OS}-${Arch}"
                    Write-Host "✓ Created $ArchiveName" -ForegroundColor Green
                } else {
                    Write-Host "⚠ tar not available, skipping ${ArchiveName}" -ForegroundColor Yellow
                }
            }
            Pop-Location

            return $true
        } else {
            Write-Host "✗ Failed to build ${OS}/${Arch}" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host "✗ Error building ${OS}/${Arch}: $_" -ForegroundColor Red
        return $false
    }
}

# Build for all platforms
Write-Host ""
Write-Host "Starting builds..." -ForegroundColor Yellow
Write-Host ""

$BuildResults = @()

# Windows (always build unless on non-Windows and specific flags set)
$BuildResults += Build-Platform "windows" "amd64"
$BuildResults += Build-Platform "windows" "arm64"

if (-not $WindowsOnly) {
    # Linux
    if (-not $SkipLinux) {
        $BuildResults += Build-Platform "linux" "amd64"
        $BuildResults += Build-Platform "linux" "arm64"
        $BuildResults += Build-Platform "linux" "arm"
    }

    # macOS
    if (-not $SkipMacOS) {
        $BuildResults += Build-Platform "darwin" "amd64"
        $BuildResults += Build-Platform "darwin" "arm64"
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
$SuccessCount = ($BuildResults | Where-Object { $_ -eq $true }).Count
$TotalCount = $BuildResults.Count
Write-Host "Build completed: $SuccessCount/$TotalCount successful" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Binaries location: ${BuildDir}\" -ForegroundColor Cyan
Write-Host ""

# List created archives
if (Test-Path $BuildDir) {
    Get-ChildItem -Path $BuildDir -Filter "*.zip" | ForEach-Object {
        $Size = [math]::Round($_.Length / 1MB, 2)
        Write-Host "  $($_.Name) ($Size MB)" -ForegroundColor Cyan
    }
    Get-ChildItem -Path $BuildDir -Filter "*.tar.gz" | ForEach-Object {
        $Size = [math]::Round($_.Length / 1MB, 2)
        Write-Host "  $($_.Name) ($Size MB)" -ForegroundColor Cyan
    }
}
