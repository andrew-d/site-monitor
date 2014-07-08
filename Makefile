# Determine whether we're being verbose or not
export V = false
export CMD_PREFIX   = @
export NULL_REDIR = 2>/dev/null >/dev/null
ifeq ($(V),true)
	CMD_PREFIX =
	NULL_REDIR =
endif

# Default build type is debug.
TYPE ?= debug

# We ignore the ".map" file in release mode, to keep the size of our binary
# to a minimum.  In debug mode, we also always load the assets from disk.
ifeq ($(TYPE),release)
export ASSET_FLAGS   :=
export BINDATA_FLAGS := "-ignore=.*\\.map"
export WEBPACK_FLAGS := --optimize-minimize --optimize-dedupe
export NODE_ENV      := NODE_ENV=production
else
export ASSET_FLAGS   := --debug
export BINDATA_FLAGS := -debug
export WEBPACK_FLAGS :=
export NODE_ENV      :=
endif

JS_FILES     := $(shell find static/js/ -name '*.js' -or -name '*.jsx')
STATIC_FILES := static/js/lib/bootstrap.min.js \
                static/css/site-monitor.css \
                static/css/bootstrap.min.css \
                static/fonts/glyphicons-halflings-regular.woff \
                static/fonts/glyphicons-halflings-regular.ttf
BUILD_FILES  := $(patsubst static/%,build/%,$(STATIC_FILES))
RESOURCES    := build/index.html build/js/bundle.js build/js/bundle.js.map $(BUILD_FILES)

WEBPACK_BIN  := ./node_modules/webpack/bin/webpack.js

# Disable all built-in rules.
.SUFFIXES:

RED     := \e[0;31m
GREEN   := \e[0;32m
YELLOW  := \e[0;33m
NOCOLOR := \e[0m


######################################################################

all: dependencies site-monitor

site-monitor: asset_descriptors.go resources.go $(wildcard *.go)
	@printf "  $(GREEN)GO$(NOCOLOR)       $@\n"
	$(CMD_PREFIX)godep go build -o $@ .

asset_descriptors.go: $(RESOURCES)
	@printf "  $(GREEN)ASSETS$(NOCOLOR)   $@\n"
	$(CMD_PREFIX)python ./gen_descriptors.py \
		$(ASSET_FLAGS) \
		--prefix "build/" \
		$(RESOURCES) > $@

resources.go: $(RESOURCES)
	@printf "  $(GREEN)BINDATA$(NOCOLOR)  $@\n"
	$(CMD_PREFIX)go-bindata \
		-ignore=\\.gitignore \
		$(BINDATA_FLAGS) \
		-prefix "./build" \
		-o $@ \
		$(sort $(dir $^))

build/js/bundle.js: $(JS_FILES)
	@printf "  $(GREEN)WEBPACK$(NOCOLOR)  $@\n"
	$(CMD_PREFIX)$(NODE_ENV) node $(WEBPACK_BIN) \
		--progress --colors $(WEBPACK_FLAGS) $(NULL_REDIR)

build/index.html: static/index.html
	@printf "  $(GREEN)CP$(NOCOLOR)       $< ==> $@\n"
	$(CMD_PREFIX)cp $< $@

build/js/%: static/js/%
	@printf "  $(GREEN)CP$(NOCOLOR)       $< ==> $@\n"
	@mkdir -p $(dir $@)
	$(CMD_PREFIX)cp $< $@

build/css/%: static/css/%
	@printf "  $(GREEN)CP$(NOCOLOR)       $< ==> $@\n"
	@mkdir -p $(dir $@)
	$(CMD_PREFIX)cp $< $@

build/fonts/%: static/fonts/%
	@printf "  $(GREEN)CP$(NOCOLOR)       $< ==> $@\n"
	@mkdir -p $(dir $@)
	$(CMD_PREFIX)cp $< $@


# This is a phony target that checks to ensure our various dependencies are installed
.PHONY: dependencies
dependencies:
	@command -v go-bindata >/dev/null 2>&1 || { printf >&2 "go-bindata is not installed, exiting...\n"; exit 1; }
	@command -v webpack    >/dev/null 2>&1 || { printf >&2 "webpack is not installed, exiting...\n"; exit 1; }
	@command -v godep      >/dev/null 2>&1 || { printf >&2 "godep is not installed, exiting...\n"; exit 1; }
	@command -v node       >/dev/null 2>&1 || { printf >&2 "node.js is not installed, exiting...\n"; exit 1; }
	@test -d node_modules/webpack    || { printf >&2 "npm dependencies not satisfied, exiting...\n"; exit 1; }
	@# Since webpack doesn't seem to exit with an error if this isn't present...
	@test -d node_modules/jsx-loader || { printf >&2 "npm dependencies not satisfied, exiting...\n"; exit 1; }

######################################################################

.PHONY: clean
CLEAN_FILES := build/index.html build/js build/css site-monitor resources.go asset_descriptors.go
clean:
	@printf "  $(YELLOW)RM$(NOCOLOR)       $(CLEAN_FILES)\n"
	$(CMD_PREFIX)$(RM) -r $(CLEAN_FILES)

.PHONY: env
env:
	@echo "JS_FILES     = $(JS_FILES)"
	@echo "STATIC_FILES = $(STATIC_FILES)"
	@echo "BUILD_FILES  = $(BUILD_FILES)"
