# Build binary
.PHONY: build
build: deps
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o $(BINARY_NAME) .

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

# Deployment command (example for a remote server)
.PHONY: deploy
deploy: docker-build docker-push
	ssh root@142.93.170.197 'docker pull sukalov/karaokebot:latest && docker restart karaokebot'