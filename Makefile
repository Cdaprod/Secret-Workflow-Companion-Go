# Makefile for Secret-Workflow-Companion-Go

BINARY_NAME=ghm
GO=go
INSTALL_DIR=/usr/local/bin
AUTOCOMPLETE_SCRIPT=ghm-autocompletion.sh
BASH_PROFILE=$(HOME)/.bash_profile
ZSH_PROFILE=$(HOME)/.zshrc

.PHONY: all build test clean run install uninstall

all: build

build:
	$(GO) mod tidy
	$(GO) build -o $(BINARY_NAME) main.go
	
test:
	$(GO) test -v ./...

clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

run:
	$(GO) run main.go

install: build
	sudo mkdir -p $(INSTALL_DIR)
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	sudo cp $(AUTOCOMPLETE_SCRIPT) $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)
	@echo "# ghm autocomplete" >> $(BASH_PROFILE)
	@echo "source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)" >> $(BASH_PROFILE)
	@echo "# ghm autocomplete" >> $(ZSH_PROFILE)
	@echo "source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)" >> $(ZSH_PROFILE)
	@echo "Installation complete. Please run 'source $(BASH_PROFILE)' or 'source $(ZSH_PROFILE)' to enable autocomplete."

uninstall:
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	sudo rm -f $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)
	sed -i '/# ghm autocomplete/d' $(BASH_PROFILE)
	sed -i "\|source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)|d" $(BASH_PROFILE)
	sed -i '/# ghm autocomplete/d' $(ZSH_PROFILE)
	sed -i "\|source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)|d" $(ZSH_PROFILE)
	@echo "Uninstallation complete."