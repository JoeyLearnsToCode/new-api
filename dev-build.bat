@REM build web

cd web
bun run build
cd ..

@REM build go

go build
