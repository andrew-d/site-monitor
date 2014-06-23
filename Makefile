# Determine whether we're being verbose or not
export V = false
export CMD_PREFIX   = @
export NULL_REDIR = 2>/dev/null >/dev/null
ifeq ($(V),true)
	CMD_PREFIX   =
	NULL_REDIR =
endif

# Default build type is debug.
TYPE ?= debug

# We ignore the ".map" file in release mode, to keep the size of our binary
# to a minimum.  In debug mode, we also always load the assets from disk.
ifeq ($(TYPE),release)
export BINDATA_FLAGS := "-ignore=.*\\.map"
else
export BINDATA_FLAGS := -debug
endif

JS_FILES     := $(shell find static/js/ -name '*.js')
STATIC_FILES := static/js/lib/bootstrap.min.js \
                static/css/bootstrap.min.css
BUILD_FILES  := $(patsubst static/%,build/%,$(STATIC_FILES))
RESOURCES    := build/index.html build/js/bundle.js $(BUILD_FILES)

# Disable all built-in rules.
.SUFFIXES:

RED     := \e[0;31m
GREEN   := \e[0;32m
YELLOW  := \e[0;33m
NOCOLOR := \e[0m


######################################################################

all: site-monitor

site-monitor: resources.go $(wildcard *.go)
	@echo -e "  $(GREEN)GO$(NOCOLOR)       $@"
	$(CMD_PREFIX)go build -o $@ .

resources.go: $(RESOURCES)
	@echo -e "  $(GREEN)BINDATA$(NOCOLOR)  $@"
	$(CMD_PREFIX)go-bindata \
		-ignore=\\.gitignore \
		$(BINDATA_FLAGS) \
		-prefix "./build" \
		-o $@ \
		$(sort $(dir $^))

build/js/bundle.js: $(JS_FILES)
	@echo -e "  $(GREEN)WEBPACK$(NOCOLOR)  $@"
	$(CMD_PREFIX)webpack --progress --colors $(NULL_REDIR)

build/index.html: static/index.html
	@echo -e "  $(GREEN)CP$(NOCOLOR)       $< ==> $@"
	$(CMD_PREFIX)cp $< $@

build/js/%: static/js/%
	@echo -e "  $(GREEN)CP$(NOCOLOR)       $< ==> $@"
	@mkdir -p $(dir $@)
	$(CMD_PREFIX)cp $< $@

build/css/%: static/css/%
	@echo -e "  $(GREEN)CP$(NOCOLOR)       $< ==> $@"
	@mkdir -p $(dir $@)
	$(CMD_PREFIX)cp $< $@


######################################################################

.PHONY: clean
CLEAN_FILES := build/index.html build/js build/css site-monitor
clean:
	@echo -e "  $(YELLOW)RM$(NOCOLOR)       $(CLEAN_FILES)"
	$(CMD_PREFIX)$(RM) -r $(CLEAN_FILES)

.PHONY: env
env:
	@echo "JS_FILES     = $(JS_FILES)"
	@echo "STATIC_FILES = $(STATIC_FILES)"
	@echo "BUILD_FILES  = $(BUILD_FILES)"
