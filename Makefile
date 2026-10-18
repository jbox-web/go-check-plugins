PLUGINS := $(wildcard check-*)

.PHONY: build test clean $(PLUGINS)

build test clean:
	@for plugin in $(PLUGINS); do $(MAKE) -C $$plugin $@ || exit 1; done
