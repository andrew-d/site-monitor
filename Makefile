# Determine whether we're being verbose or not
export V = false
export CMD_PREFIX   = @
export OUTPUT_REDIR = > /dev/null
ifeq ($(V),true)
	CMD_PREFIX   =
	OUTPUT_REDIR =
endif

# Default build type is debug.
TYPE ?= debug

# Figure out what type of build we're doing, and properly set flags and output
# locations.
ifeq ($(TYPE),release)
export BINDATA_FLAGS :=
else
export BINDATA_FLAGS := -debug
endif


JS_FILES := $(shell find static/js/ -name '*.js')

# Disable all built-in rules.
.SUFFIXES:


######################################################################

all: site-monitor

site-monitor: resources.go
	$(CMD_PREFIX)go build -o $@ .

resources.go: build/bundle.js build/index.html
	$(CMD_PREFIX)go-bindata -ignore=\\.gitignore -ignore=site-monitor $(BINDATA_FLAGS) \
		-o $@ -prefix "./build" ./build

build/index.html: static/index.html
	$(CMD_PREFIX)cp $< $@

build/bundle.js: $(JS_FILES)
	$(CMD_PREFIX)webpack --progress --colors $(OUTPUT_REDIR)


######################################################################

.PHONY: clean
clean:
	$(RM) build/bundle.* build/index.html site-monitor
