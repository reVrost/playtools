# Makefile for github.com/revrost/playtools

# Variables
APP_NAME=playtools
GITHUB_REPO=github.com/revrost/playtools
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
GOBUILD=go build -o $(APP_NAME) main.go

.PHONY: build release

build:
	$(GOBUILD)
	@echo "Build complete: ./$(APP_NAME)"

release:
	@echo "Current version: $(VERSION)"
	@read -p "Enter version increment (major/minor/patch): " increment; \
	case "$$increment" in \
		major) \
			new_version=$$(semver -i major $(VERSION)); \
			;; \
		minor) \
			new_version=$$(semver -i minor $(VERSION)); \
			;; \
		patch) \
			new_version=$$(semver -i patch $(VERSION)); \
			;; \
		*) \
			echo "Invalid increment type. Use major, minor, or patch."; \
			exit 1; \
			;; \
	esac; \
	echo "New version: $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	$(GOBUILD); \
	echo "Release $$new_version created successfully!"

