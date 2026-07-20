FRONTEND_DIR = ./web
BACKEND_DIR = .

.PHONY: all build-frontend start-backend

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && BUILD_TIME=$$(TZ=Asia/Shanghai date '+%Y-%m-%d %H:%M:%S') && echo "Build time: $$BUILD_TIME" && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && BUILD_TIME=$$(TZ=Asia/Shanghai date '+%Y-%m-%d %H:%M:%S') && go run -ldflags "-X 'github.com/QuantumNous/new-api/common.BuildTime=$$BUILD_TIME'" main.go &
