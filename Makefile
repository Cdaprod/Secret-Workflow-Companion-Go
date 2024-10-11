# Makefile for Secret-Workflow-Companion-Go

BINARY_NAME=ghm
GO=go
INSTALL_DIR=/usr/local/bin  # System-wide installation in /usr/local/bin
AUTOCOMPLETE_SCRIPT=ghm-autocompletion.sh  # Located at the root of the repo
BASH_PROFILE=$(HOME)/.bash_profile
ZSH_PROFILE=$(HOME)/.zshrc

.PHONY: all build clean run test install uninstall

all: build

build:
	$(GO) build -o $(BINARY_NAME) main.go

clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

run:
	$(GO) run main.go

test:
	$(GO) test ./...

install: build
	sudo mkdir -p $(INSTALL_DIR)  # Ensure /usr/local/bin exists
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/  # Corrected: copy the binary to /usr/local/bin
	sudo cp $(AUTOCOMPLETE_SCRIPT) $(INSTALL_DIR)/  # Copy the autocompletion script to /usr/local/bin
	@echo "# ghm autocomplete" >> $(BASH_PROFILE)
	@echo "source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)" >> $(BASH_PROFILE)
	@echo "# ghm autocomplete" >> $(ZSH_PROFILE)
	@echo "source $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)" >> $(ZSH_PROFILE)
	@echo "Installation complete. Please run 'source $(BASH_PROFILE)' (for Bash) or 'source $(ZSH_PROFILE)' (for Zsh) to enable autocomplete."

uninstall:
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	sudo rm -f $(INSTALL_DIR)/$(AUTOCOMPLETE_SCRIPT)
	sed -i '/# ghm autocomplete/d' $(BASH_PROFILE)
	sed -i '/source.*ghm-autocompletion\.sh/d' $(BASH_PROFILE)
	sed -i '/# ghm autocomplete/d' $(ZSH_PROFILE)
	sed -i '/source.*ghm-autocompletion\.sh/d' $(ZSH_PROFILE)
	@echo "Uninstallation complete."