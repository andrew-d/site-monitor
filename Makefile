

all: site-monitor


site-monitor: resources.go
	go build -o $@ .

resources.go: build/bundle.js build/index.html
	go-bindata -o $@ -ignore=\\.gitignore -ignore=site-monitor -prefix "./build" ./build

build/index.html: static/index.html
	cp $< $@


JS_FILES := $(shell find static/js/ -name '*.js')

build/bundle.js: $(JS_FILES)
	webpack --progress --colors
