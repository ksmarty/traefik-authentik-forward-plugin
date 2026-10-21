.POSIX:

$(V).SILENT:

.PHONY: vendor clean lint test yaegi_test e2e_test hash

PACKAGES := $(shell go list ./...)

default: lint test

vendor:
	go mod vendor

clean:
	rm -rf ./vendor

lint:
	golangci-lint run

test:
	go test -v -cover ./...

# hash prints the value Traefik compares against for a released version; see
# scripts/plugin-hash.sh for why it is not stable across registry repacks.
hash:
	./scripts/plugin-hash.sh $(VERSION)

yaegi_test:
	$(foreach pkg, $(PACKAGES), yaegi test -v $(pkg);)

e2e_test:
	cd e2e && npm install --include=dev && npm run test:install && npm run test

e2e_ui_test:
	cd e2e && npm install --include=dev && npm run test:install &&  npm run test:dev
