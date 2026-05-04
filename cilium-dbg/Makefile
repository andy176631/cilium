# Copyright Authors of Cilium
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := all

ROOT_DIR := $(shell dirname "$(realpath $(lastword $(MAKEFILE_LIST)))")

include ${ROOT_DIR}/../Makefile.defs

# Add the ability to override variables
# ROOT_DIR changes to repo root after including Makefile.defs
-include ${ROOT_DIR}/Makefile.override

TARGET := cilium-dbg
TARGET_OLD := cilium
CMDREF_FLAGS_SRC_DIR := $(ROOT_DIR)/Documentation/cmdref/flags
CMDREF_FLAGS_OUT_DIR := cmd/assets
CMDREF_FLAG_FILES := $(notdir $(wildcard $(CMDREF_FLAGS_SRC_DIR)/*.yaml))
CMDREF_FLAG_TARGETS := $(addprefix $(CMDREF_FLAGS_OUT_DIR)/, $(CMDREF_FLAG_FILES))
$(CMDREF_FLAGS_OUT_DIR)/%.yaml: $(CMDREF_FLAGS_SRC_DIR)/%.yaml
	@cp $< $@

.PHONY: all $(TARGET) clean install

all: $(TARGET)

$(TARGET): $(CMDREF_FLAG_TARGETS)
	@$(ECHO_GO)
	$(QUIET)$(GO_BUILD) -o $@

clean:
	@$(ECHO_CLEAN)
	-$(QUIET)rm -f $(TARGET) $(CMDREF_FLAG_TARGETS)
	$(QUIET)$(GO_CLEAN)

install: install-binary install-bash-completion-only

install-binary:
	$(QUIET)$(INSTALL) -m 0755 -d $(DESTDIR)$(BINDIR)
	$(QUIET)$(INSTALL) -m 0755 $(TARGET) $(DESTDIR)$(BINDIR)
	$(QUIET)ln -frs $(DESTDIR)$(BINDIR)/$(TARGET) $(DESTDIR)$(BINDIR)/$(TARGET_OLD)

install-bash-completion: $(TARGET) install-bash-completion-only

install-bash-completion-only:
	$(QUIET)$(INSTALL) -m 0755 -d $(DESTDIR)$(CONFDIR)/bash_completion.d
	./$(TARGET) completion bash > $(TARGET)_bash_completion
	$(QUIET)$(INSTALL) -m 0644 -T $(TARGET)_bash_completion $(DESTDIR)$(CONFDIR)/bash_completion.d/$(TARGET)
