.PHONY: build run test frontend-install frontend-build clean dev

build:
	go build -o bin/server ./cmd/server/

run: build
	./bin/server

test:
	go test ./...

frontend-install:
	cd web && npm install

frontend-build: frontend-install
	cd web && npm run build

clean:
	rm -rf bin/ web/node_modules web/dist

dev:
	@echo "Starting backend and frontend in parallel..."
	@trap 'kill 0' INT; \
	go run ./cmd/server/ & \
	cd web && npm run dev & \
	wait
