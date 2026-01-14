@echo off
REM ============================================================================
REM NodeByte Database Initialization Script (Windows)
REM ============================================================================
REM This script initializes the complete NodeByte database schema
REM Usage: init-database.bat <DATABASE_URL>
REM
REM Example:
REM   init-database.bat "postgresql://user:password@localhost:5432/nodebyte"
REM
REM ============================================================================

setlocal enabledelayedexpansion

if "%~1"=="" (
    echo Usage: %0 ^<DATABASE_URL^>
    echo.
    echo Example:
    echo   %0 "postgresql://user:password@localhost:5432/nodebyte"
    echo.
    exit /b 1
)

set "DATABASE_URL=%~1"
set "SCRIPT_DIR=%~dp0"

echo ============================================================================
echo NodeByte Database Initialization
echo ============================================================================
echo.
echo Database URL: %DATABASE_URL% (password hidden^)
echo Schema directory: %SCRIPT_DIR%
echo.

REM Array of schema files in execution order
set "SCHEMAS[0]=schema_01_users_auth.sql"
set "SCHEMAS[1]=schema_02_pterodactyl_sync.sql"
set "SCHEMAS[2]=schema_03_servers.sql"
set "SCHEMAS[3]=schema_04_billing.sql"
set "SCHEMAS[4]=schema_05_support_tickets.sql"
set "SCHEMAS[5]=schema_06_discord_webhooks.sql"
set "SCHEMAS[6]=schema_07_sync_logs.sql"
set "SCHEMAS[7]=schema_08_config.sql"
set "SCHEMAS[8]=schema_09_hytale.sql"

REM Execute each schema file
for /L %%i in (0,1,8) do (
    set "schema=!SCHEMAS[%%i]!"
    set "schema_path=%SCRIPT_DIR%!schema!"
    
    if not exist "!schema_path!" (
        echo ❌ Schema file not found: !schema_path!
        exit /b 1
    )
    
    echo 📦 Executing: !schema!
    psql "%DATABASE_URL%" -f "!schema_path!" > nul 2>&1
    if !errorlevel! equ 0 (
        echo ✅ !schema!
    ) else (
        echo ❌ Failed to execute !schema!
        exit /b 1
    )
)

echo.
echo ============================================================================
echo ✅ Database initialization complete!
echo ============================================================================
echo.
echo Summary:
echo   - Users ^& Authentication (users, sessions, password_reset_tokens, verification_tokens^)
echo   - Pterodactyl Sync (locations, nodes, allocations, nests, eggs, egg_variables, egg_properties^)
echo   - Servers (servers, server_variables, server_properties, server_databases, server_backups^)
echo   - Billing (products, invoices, invoice_items, payments^)
echo   - Support (support_tickets, support_ticket_replies^)
echo   - Discord Webhooks (discord_webhooks^)
echo   - Sync Logs (sync_logs^)
echo   - Config (config^)
echo   - Hytale OAuth (hytale_oauth_tokens, hytale_game_sessions^)
echo.
echo You can now start your backend with:
echo   go run ./cmd/api/main.go
echo.

endlocal
