.PHONY: dev backend frontend build clean install

# Run both backend and frontend for local development
dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait

# Run the Go backend
backend:
	cd backend && go run ./cmd/kcogs

# Run the React frontend dev server
frontend:
	cd frontend && npm run dev

# Install dependencies
install:
	cd frontend && npm install
	cd backend && go mod download

# Build for production
build:
	cd backend && go build -o bin/kcogs ./cmd/kcogs
	cd frontend && npm run build

# Clean build artifacts
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
