@REM build local binary
for /f %%i in ('powershell -NoProfile -Command "([System.TimezoneInfo]::ConvertTimeBySystemTimeZoneId((Get-Date), 'China Standard Time')).ToString('yyyy-MM-dd HH:mm:ss')"') do set BUILD_TIME=%%i
go build -ldflags "-X 'github.com/QuantumNous/new-api/common.BuildTime=%BUILD_TIME%' -X 'github.com/QuantumNous/new-api/common.Version=%BUILD_TIME%'"

@REM build freebsd binary

set GOOS=freebsd
set GOARCH=amd64
for /f %%i in ('powershell -NoProfile -Command "([System.TimezoneInfo]::ConvertTimeBySystemTimeZoneId((Get-Date), 'China Standard Time')).ToString('yyyy-MM-dd HH:mm:ss')"') do set BUILD_TIME=%%i
go build -ldflags "-X 'github.com/QuantumNous/new-api/common.BuildTime=%BUILD_TIME%' -X 'github.com/QuantumNous/new-api/common.Version=%BUILD_TIME%'" -o one-api-freebsd
