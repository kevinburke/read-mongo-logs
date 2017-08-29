SHELL = /bin/bash

BUMP_VERSION := $(GOPATH)/bin/bump_version
MEGACHECK := $(GOPATH)/bin/megacheck
RELEASE := $(GOPATH)/bin/github-release

test: lint
	go test ./...

$(MEGACHECK):
	go get honnef.co/go/tools/cmd/megacheck

lint: $(MEGACHECK)
	$(MEGACHECK) --ignore='github.com/kevinburke/read-mongo-logs/*.go:U1000' ./...
	go vet ./...

race-test: lint
	go test -race ./...

$(RELEASE):
	go get -u github.com/aktau/github-release

$(BUMP_VERSION):
	go get github.com/Shyp/bump_version

# Run "GITHUB_TOKEN=my-token make release version=0.x.y" to release a new version.
release: race-test | $(BUMP_VERSION) $(RELEASE)
ifndef version
	@echo "Please provide a version"
	exit 1
endif
ifndef GITHUB_TOKEN
	@echo "Please set GITHUB_TOKEN in the environment"
	exit 1
endif
	$(BUMP_VERSION) --version=$(version) main.go
	git push origin --tags
	mkdir -p releases/$(version)
	# Change the binary names below to match your tool name
	GOOS=linux GOARCH=amd64 go build -o releases/$(version)/read-mongo-logs-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o releases/$(version)/read-mongo-logs-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o releases/$(version)/read-mongo-logs-windows-amd64 .
	# Change the Github username to match your username.
	# These commands are not idempotent, so ignore failures if an upload repeats
	$(RELEASE) release --user kevinburke --repo read-mongo-logs --tag $(version) || true
	$(RELEASE) upload --user kevinburke --repo read-mongo-logs --tag $(version) --name read-mongo-logs-linux-amd64 --file releases/$(version)/read-mongo-logs-linux-amd64 || true
	$(RELEASE) upload --user kevinburke --repo read-mongo-logs --tag $(version) --name read-mongo-logs-darwin-amd64 --file releases/$(version)/read-mongo-logs-darwin-amd64 || true
	$(RELEASE) upload --user kevinburke --repo read-mongo-logs --tag $(version) --name read-mongo-logs-windows-amd64 --file releases/$(version)/read-mongo-logs-windows-amd64 || true
