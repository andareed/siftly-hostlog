APP      := sfhost
VERSION  ?= $(shell git describe --tags --always --dirty)
LDFLAGS  := -X 'main.Version=$(VERSION)'
DIST_DIR := dist

.PHONY: all clean linux windows mac mac-amd64 mac-arm64 release

all: linux windows mac

clean:
	rm -rf $(DIST_DIR)

linux:
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP)_$(VERSION)_linux_amd64 .

windows:
	mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP)_$(VERSION)_windows_amd64.exe .

mac: mac-amd64 mac-arm64

mac-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP)_$(VERSION)_darwin_amd64 .

mac-arm64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP)_$(VERSION)_darwin_arm64 .

release: clean all
	@echo "Built release binaries in $(DIST_DIR):"
	@ls -1 $(DIST_DIR)
