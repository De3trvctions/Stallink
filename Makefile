.PHONY: test dev nacos-publish reset-local seed-local web compose

test dev nacos-publish reset-local seed-local:
	$(MAKE) -C server $@

web:
	cd web && npm run dev

compose:
	docker compose up -d
