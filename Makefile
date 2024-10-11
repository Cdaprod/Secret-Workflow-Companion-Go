# Makefile for Secret Workflow Companion (Golang)

BINARY_NAME=ghmanager
GO=go
INSTALL_DIR=$(HOME)/bin
AUTOCOMPLETE_SCRIPT=scripts/autocompletion.sh
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
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_NAME) $(INSTALL_DIR)/
	cp $(AUTOCOMPLETE_SCRIPT) $(INSTALL_DIR)/
	@echo "# ghmanager autocomplete" >> $(BASH_PROFILE)
	@echo "source $(INSTALL_DIR)/autocompletion.sh" >> $(BASH_PROFILE)
	@echo "# ghmanager autocomplete" >> $(ZSH_PROFILE)
	@echo "source $(INSTALL_DIR)/autocompletion.sh" >> $(ZSH_PROFILE)
	@echo "Installation complete. Please run 'source $(BASH_PROFILE)' (for Bash) or 'source $(ZSH_PROFILE)' (for Zsh) to enable autocomplete."

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -f $(INSTALL_DIR)/autocompletion.sh
	sed -i '/# ghmanager autocomplete/d' $(BASH_PROFILE)
	sed -i '/source.*autocompletion\.sh/d' $(BASH_PROFILE)
	sed -i '/# ghmanager autocomplete/d' $(ZSH_PROFILE)
	sed -i '/source.*autocompletion\.sh/d' $(ZSH_PROFILE)
	@echo "Uninstallation complete. Please restart your terminal or run 'source $(BASH_PROFILE)' (for Bash) or 'source $(ZSH_PROFILE)' (for Zsh) to apply changes."