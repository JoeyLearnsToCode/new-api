@REM build local binary

go build

@REM build freebsd binary

set GOOS=freebsd
set GOARCH=amd64
go build -o one-api-freebsd
