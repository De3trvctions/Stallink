.PHONY: test dev nacos-publish reset-local seed-local

test:
	go test ./...

dev:
	go run ./cmd/stallink -conf configs/local/stallink.yaml

reset-local:
	@set -eu; \
	stamp=$$(date +%Y%m%d%H%M%S); \
	db_backup="/private/tmp/stallink-pre-reset-$$stamp.dump"; \
	blobs_backup="/private/tmp/stallink-pre-reset-$$stamp-blobs.tar.gz"; \
	docker exec postgres pg_dump -U postgres -Fc -d stallink > "$$db_backup"; \
	tar -czf "$$blobs_backup" -C data blobs; \
	chmod 600 "$$db_backup" "$$blobs_backup"; \
	go run ./cmd/stallink reset-local -conf configs/local/stallink.yaml -confirm stallink -backup "$$db_backup" -blobs-backup "$$blobs_backup"

seed-local:
	go run ./cmd/stallink seed-local -conf configs/local/stallink.yaml

nacos-publish:
	@set -eu; \
	: "$${STALLINK_NACOS_USERNAME:?set STALLINK_NACOS_USERNAME for local Nacos authentication}"; \
	: "$${STALLINK_NACOS_PASSWORD:?set STALLINK_NACOS_PASSWORD for local Nacos authentication}"; \
	token=$$(curl -fsS -X POST 'http://127.0.0.1:8848/nacos/v1/auth/login' \
	  --data-urlencode "username=$$STALLINK_NACOS_USERNAME" \
	  --data-urlencode "password=$$STALLINK_NACOS_PASSWORD" \
	  | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])'); \
	curl -sS -X POST "http://127.0.0.1:8848/nacos/v1/console/namespaces?accessToken=$$token" -d 'customNamespaceId=stallink&namespaceName=stallink&namespaceDesc=Stallink'; \
	echo; \
	curl -sS -X POST "http://127.0.0.1:8848/nacos/v1/cs/configs?accessToken=$$token" \
	  --data-urlencode "dataId=stallink.yaml" \
	  --data-urlencode "group=DEFAULT_GROUP" \
	  --data-urlencode "tenant=stallink" \
	  --data-urlencode "type=yaml" \
	  --data-urlencode "content@configs/nacos/stallink.yaml"; \
	echo
