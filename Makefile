.PHONY: all clean spoke spoke-server sprocket build-all

BINDIR := bin
CMDDIR := cmd

all: build-all

build-all: spoke spoke-server sprocket

spoke-server:
	@echo "Building spoke server..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/spoke-server $(CMDDIR)/spoke/main.go

spoke:
	@echo "Building spoke-cli tool..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/spoke $(CMDDIR)/spoke-cli/main.go

sprocket:
	@echo "Building sprocket watcher..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/sprocket $(CMDDIR)/sprocket/*.go

clean:
	@echo "Cleaning up..."
	rm -rf $(BINDIR)

test:
	go test -v ./...

.DEFAULT_GOAL := all 