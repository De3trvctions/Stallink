.PHONY: test dev nacos-publish reset-local seed-local web

test dev nacos-publish reset-local seed-local:
	$(MAKE) -C server $@

web:
	cd web && npm run dev
