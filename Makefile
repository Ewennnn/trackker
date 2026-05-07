.PHONY: build clean install run help

BINARY_NAME := trackker.exe
WEB_DIR := web-controls
CMD_DIR := ./cmd/api

# -------------------
# HELP
# -------------------

help:
	@echo "🚀 Trackker build system"
	@echo ""
	@echo "make build   -> full production build (clean + frontend + backend)"
	@echo "make clean   -> clean artifacts"
	@echo "make run     -> build + run"
	@echo "make install -> install frontend deps"

# -------------------
# INSTALL FRONTEND
# -------------------

install:
	@echo "📦 Installing frontend dependencies..."
	cd $(WEB_DIR) && npm install

# -------------------
# CLEAN
# -------------------

clean:
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf $(WEB_DIR)/dist
	rm -rf $(WEB_DIR)/node_modules
	go clean ./...

# -------------------
# BUILD FRONTEND
# -------------------

build-frontend: install
	@echo "🏗️ Building frontend..."
	cd $(WEB_DIR) && npm run build:prod

# -------------------
# BUILD BACKEND (EMBED FRONTEND)
# -------------------

build-backend:
	@echo "🏗️ Building backend (Go + embedded frontend)..."
	go build -o $(BINARY_NAME) $(CMD_DIR)

# -------------------
# FULL BUILD (PROD)
# -------------------

build: clean build-frontend build-backend
	@echo ""
	@echo "✅ Build complete!"
	@echo "Binary: $(BINARY_NAME)"

# -------------------
# RUN
# -------------------

run: build
	@echo "🚀 Running..."
	@./$(BINARY_NAME)