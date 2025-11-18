@echo off
REM vaultctl - Build and Run Script for Windows
REM This script helps build and run the vaultctl password manager

setlocal enabledelayedexpansion

REM Script directory
set "SCRIPT_DIR=%~dp0"
set "BINARY_NAME=vaultctl.exe"
set "BINARY_PATH=%SCRIPT_DIR%%BINARY_NAME%"

REM Main script logic
if "%1"=="" goto no_command
if "%1"=="build" goto build
if "%1"=="run" goto run
if "%1"=="build-and-run" goto build_and_run
if "%1"=="buildrun" goto build_and_run
if "%1"=="clean" goto clean
if "%1"=="help" goto help
if "%1"=="--help" goto help
if "%1"=="-h" goto help
goto unknown_command

:build
call :check_prerequisites
call :build_application
goto end

:run
shift
call :run_application %*
goto end

:build_and_run
call :check_prerequisites
call :build_application
shift
call :run_application %*
goto end

:clean
call :clean_build
goto end

:help
call :show_usage
goto end

:no_command
call :check_prerequisites
call :check_binary_exists
if errorlevel 1 (
    echo [INFO] Binary not found. Building...
    call :build_application
)
echo.
echo [INFO] Showing vaultctl help:
echo.
"%BINARY_PATH%" --help
goto end

:unknown_command
call :check_binary_exists
if errorlevel 1 (
    echo [WARNING] Binary not found. Building first...
    call :check_prerequisites
    call :build_application
)
call :run_application %*
goto end

REM Function to check prerequisites
:check_prerequisites
echo [INFO] Checking prerequisites...
echo.

set "MISSING=0"

where go >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed or not in PATH
    echo [INFO] Please install Go 1.23 or later from https://go.dev
    set "MISSING=1"
) else (
    for /f "tokens=3" %%i in ('go version') do set "GO_VERSION=%%i"
    set "GO_VERSION=!GO_VERSION:go=!"
    echo [SUCCESS] Go is installed (version: !GO_VERSION!)
)

where aws >nul 2>&1
if errorlevel 1 (
    echo [WARNING] AWS CLI is not installed or not in PATH
    echo [INFO] DynamoDB sync will not work without AWS CLI
    echo [INFO] Install from: https://aws.amazon.com/cli/
) else (
    echo [SUCCESS] AWS CLI is installed
)

if !MISSING!==1 (
    echo.
    echo [ERROR] Missing required prerequisites. Please install them and try again.
    exit /b 1
)
echo.
exit /b 0

REM Function to build the application
:build_application
echo [INFO] Building %BINARY_NAME%...
echo.

cd /d "%SCRIPT_DIR%"

REM Download dependencies
echo [INFO] Downloading dependencies...
go mod download
if errorlevel 1 (
    echo [ERROR] Failed to download dependencies
    exit /b 1
)

REM Tidy modules
echo [INFO] Tidying modules...
go mod tidy

REM Build the application
echo [INFO] Compiling...
go build -o "%BINARY_NAME%" .
if errorlevel 1 (
    echo [ERROR] Build failed
    exit /b 1
)

echo [SUCCESS] %BINARY_NAME% built successfully!
echo [INFO] Binary location: %BINARY_PATH%
echo.
exit /b 0

REM Function to check if binary exists
:check_binary_exists
if exist "%BINARY_PATH%" (
    exit /b 0
) else (
    exit /b 1
)

REM Function to run the application
:run_application
call :check_binary_exists
if errorlevel 1 (
    echo [WARNING] Binary not found. Building first...
    call :build_application
)

echo [INFO] Running %BINARY_NAME%...
echo.

REM Pass all arguments to the binary
"%BINARY_PATH%" %*
exit /b 0

REM Function to show usage
:show_usage
echo.
echo vaultctl Build and Run Script
echo.
echo Usage: run.bat [command] [options]
echo.
echo Commands:
echo   build              Build the application
echo   run [args...]      Run the application (passes all args to vaultctl)
echo   build-and-run      Build and then run the application
echo   clean              Remove build artifacts
echo   help               Show this help message
echo.
echo Examples:
echo   run.bat build
echo   run.bat run init
echo   run.bat run add --name github --username user@example.com
echo   run.bat build-and-run list
echo   run.bat clean
echo.
echo If no command is specified, the script will:
echo   1. Check prerequisites
echo   2. Build the application if needed
echo   3. Show vaultctl help
echo.
exit /b 0

REM Function to clean build artifacts
:clean_build
echo [INFO] Cleaning build artifacts...
echo.

if exist "%BINARY_PATH%" (
    del /f "%BINARY_PATH%"
    echo [SUCCESS] Removed %BINARY_NAME%
) else (
    echo [INFO] No binary to clean
)
echo.
exit /b 0

:end
endlocal

