# vaultctl - Build and Run Script for Windows (PowerShell)
# This script helps build and run the vaultctl password manager

param(
    [Parameter(Position=0)]
    [string]$Command = "",
    
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$Arguments
)

# Script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinaryName = "vaultctl.exe"
$BinaryPath = Join-Path $ScriptDir $BinaryName

# Colors for output (PowerShell supports colors natively)
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Function to check if command exists
function Test-Command {
    param([string]$CommandName)
    $null -ne (Get-Command $CommandName -ErrorAction SilentlyContinue)
}

# Function to check prerequisites
function Check-Prerequisites {
    Write-Info "Checking prerequisites..."
    Write-Host ""
    
    $missing = $false
    
    if (-not (Test-Command "go")) {
        Write-Error "Go is not installed or not in PATH"
        Write-Info "Please install Go 1.23 or later from https://go.dev"
        $missing = $true
    } else {
        $goVersion = (go version).Split(' ')[2] -replace 'go', ''
        Write-Success "Go is installed (version: $goVersion)"
    }
    
    if (-not (Test-Command "aws")) {
        Write-Warning "AWS CLI is not installed or not in PATH"
        Write-Info "DynamoDB sync will not work without AWS CLI"
        Write-Info "Install from: https://aws.amazon.com/cli/"
    } else {
        Write-Success "AWS CLI is installed"
    }
    
    if ($missing) {
        Write-Host ""
        Write-Error "Missing required prerequisites. Please install them and try again."
        exit 1
    }
    Write-Host ""
}

# Function to build the application
function Build-Application {
    Write-Info "Building $BinaryName..."
    Write-Host ""
    
    Push-Location $ScriptDir
    
    try {
        # Download dependencies
        Write-Info "Downloading dependencies..."
        go mod download
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to download dependencies"
            exit 1
        }
        
        # Tidy modules
        Write-Info "Tidying modules..."
        go mod tidy
        
        # Build the application
        Write-Info "Compiling..."
        go build -o $BinaryName .
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Build failed"
            exit 1
        }
        
        Write-Success "$BinaryName built successfully!"
        Write-Info "Binary location: $BinaryPath"
        Write-Host ""
    } finally {
        Pop-Location
    }
}

# Function to check if binary exists
function Test-BinaryExists {
    Test-Path $BinaryPath
}

# Function to run the application
function Run-Application {
    if (-not (Test-BinaryExists)) {
        Write-Warning "Binary not found. Building first..."
        Build-Application
    }
    
    Write-Info "Running $BinaryName..."
    Write-Host ""
    
    # Pass all arguments to the binary
    & $BinaryPath $Arguments
}

# Function to show usage
function Show-Usage {
    Write-Host ""
    Write-Host "vaultctl Build and Run Script" -ForegroundColor Green
    Write-Host ""
    Write-Host "Usage: .\run.ps1 [command] [options]"
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  build              Build the application"
    Write-Host "  run [args...]      Run the application (passes all args to vaultctl)"
    Write-Host "  build-and-run      Build and then run the application"
    Write-Host "  clean              Remove build artifacts"
    Write-Host "  help               Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\run.ps1 build"
    Write-Host "  .\run.ps1 run init"
    Write-Host "  .\run.ps1 run add --name github --username user@example.com"
    Write-Host "  .\run.ps1 build-and-run list"
    Write-Host "  .\run.ps1 clean"
    Write-Host ""
    Write-Host "If no command is specified, the script will:"
    Write-Host "  1. Check prerequisites"
    Write-Host "  2. Build the application if needed"
    Write-Host "  3. Show vaultctl help"
    Write-Host ""
}

# Function to clean build artifacts
function Clean-Build {
    Write-Info "Cleaning build artifacts..."
    Write-Host ""
    
    if (Test-Path $BinaryPath) {
        Remove-Item $BinaryPath -Force
        Write-Success "Removed $BinaryName"
    } else {
        Write-Info "No binary to clean"
    }
    Write-Host ""
}

# Main script logic
switch ($Command.ToLower()) {
    "build" {
        Check-Prerequisites
        Build-Application
    }
    "run" {
        Run-Application
    }
    "build-and-run" {
        Check-Prerequisites
        Build-Application
        Run-Application
    }
    "buildrun" {
        Check-Prerequisites
        Build-Application
        Run-Application
    }
    "clean" {
        Clean-Build
    }
    "help" {
        Show-Usage
    }
    "--help" {
        Show-Usage
    }
    "-h" {
        Show-Usage
    }
    "" {
        # No command specified - check, build if needed, show help
        Check-Prerequisites
        if (-not (Test-BinaryExists)) {
            Write-Info "Binary not found. Building..."
            Build-Application
        }
        Write-Host ""
        Write-Info "Showing vaultctl help:"
        Write-Host ""
        & $BinaryPath --help
    }
    default {
        # Unknown command - try to run it as vaultctl command
        if (Test-BinaryExists) {
            $Arguments = @($Command) + $Arguments
            Run-Application
        } else {
            Write-Warning "Binary not found. Building first..."
            Check-Prerequisites
            Build-Application
            $Arguments = @($Command) + $Arguments
            Run-Application
        }
    }
}

