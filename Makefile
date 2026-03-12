VERSION ?= dev
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o go-mine .

run:
	go run .

test:
	go test -v ./...

release: clean test
	@mkdir -p dist
	cp go-mine.1 dist/
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/go-mine . && \
		tar -czf dist/go-mine-$(VERSION)-linux-amd64.tar.gz -C dist go-mine go-mine.1 && rm dist/go-mine
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/go-mine . && \
		tar -czf dist/go-mine-$(VERSION)-linux-arm64.tar.gz -C dist go-mine go-mine.1 && rm dist/go-mine
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/go-mine . && \
		tar -czf dist/go-mine-$(VERSION)-darwin-amd64.tar.gz -C dist go-mine go-mine.1 && rm dist/go-mine
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/go-mine . && \
		tar -czf dist/go-mine-$(VERSION)-darwin-arm64.tar.gz -C dist go-mine go-mine.1 && rm dist/go-mine
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/go-mine.exe . && \
		cd dist && zip go-mine-$(VERSION)-windows-amd64.zip go-mine.exe go-mine.1 && rm go-mine.exe
	rm dist/go-mine.1

clean:
	rm -rf dist/
	rm -f go-mine

deploy: build install-man install-completion
	cp go-mine ~/.local/bin/

install-man:
	install -d /usr/local/share/man/man1
	install -m 644 go-mine.1 /usr/local/share/man/man1/go-mine.1

install-completion:
	install -d ~/.oh-my-zsh/custom/completions
	install -m 644 _go-mine ~/.oh-my-zsh/custom/completions/_go-mine

.PHONY: build run test release clean deploy install-man install-completion
