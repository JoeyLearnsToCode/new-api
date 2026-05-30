FRONTEND_DIR = ./web
BACKEND_DIR = .
MOE_ATELIER_REPO = https://github.com/JoeyLearnsToCode/moe-atelier.git

.PHONY: all build-frontend build-moe-frontend start-backend build-all

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$$(cat VERSION) bun run build

build-moe-frontend:
	@echo "Building moe-atelier frontend..."
	@pwsh scripts/build-moe-atelier.ps1 -MoeAtelierRepo $(MOE_ATELIER_REPO)

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

build-all: build-frontend build-moe-frontend
	@echo "Building Go backend..."
	@cd $(BACKEND_DIR) && go build -ldflags "-s -w -X 'one-api/common.Version=$$(cat VERSION)'" -o new-api
