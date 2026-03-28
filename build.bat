@echo off
echo Building Schemata desktop app...
call wails build -o schemata.exe
if %errorlevel% neq 0 (
    echo FAILED: Desktop app build
    exit /b 1
)
echo.

echo Building MCP server...
go build -o build/bin/schemata-mcp.exe ./cmd/mcp-server/
if %errorlevel% neq 0 (
    echo FAILED: MCP server build
    exit /b 1
)
echo.

echo All builds successful.
echo   Desktop app: build\bin\schemata.exe
echo   MCP server:  build\bin\schemata-mcp.exe
