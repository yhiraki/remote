GOBINARY=remote
GOBUILD=go build

all: build

build:
	$(GOBUILD) -o $(GOBINARY)

clean:
	$(GOBUILD) clean
	rm -f $(GOBINARY)

.PHONY: all build clean
