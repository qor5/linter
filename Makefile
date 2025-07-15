OWNER = qor5
IMAGE_NAME = linter

.PHONY: build-docker

build-docker:
	@echo "Checking for git tag on current HEAD..."
	@TAG=$$(git describe --tags --exact-match HEAD 2>/dev/null) || { \
		echo "Error: No tag found on current HEAD. Please create a tag first."; \
		echo "Example: git tag v1.0.0 && git push origin v1.0.0"; \
		exit 1; \
	}; \
	echo "Building Docker image with tag: $$TAG"; \
	docker buildx build -f .docker/Dockerfile -t $(OWNER)/$(IMAGE_NAME):$$TAG .
