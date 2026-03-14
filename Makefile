.PHONY: build web dev clean

build: web
	go build -ldflags "-s -w -X main.version=dev -X main.commit=$$(git rev-parse --short HEAD)" -o proxpilot .

web:
	cd web && npm ci && npm run build

dev:
	cd web && npm run dev

clean:
	rm -rf proxpilot web/dist web/node_modules
