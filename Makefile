# Define variables
BINARY_NAME=bin/karaokebot
LDFLAGS=

# Download dependencies
.PHONY: deps
deps:
	go mod download

# Build binary
.PHONY: build
build: deps
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/karaokebot

# Build Docker image
.PHONY: docker-build
docker-build: build
	docker buildx build -t sukalov/karaokebot --platform linux/amd64 .

# Push Docker image
.PHONY: docker-push
docker-push:
	docker push sukalov/karaokebot:latest

# Development run with Air hot reload
.PHONY: dev
dev:
	air

# Clean up old Docker images
.PHONY: docker-clean
docker-clean:
	ssh root@142.93.170.197 "\
		docker stop karaoke || true; \
		docker rm karaoke || true; \
		docker image prune -f \
	"

# Deployment command
.PHONY: deploy
deploy: build docker-build docker-push docker-clean
	ssh root@142.93.170.197 "\
		docker pull sukalov/karaokebot:latest; \
		docker run --name karaoke \
		--restart always \
		--env-file .env -v \
		$(pwd)/root/.env:/root/.env \
		-d sukalov/karaokebot:latest \
	"