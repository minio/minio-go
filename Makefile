all: checks

.PHONY: examples docs

checks: vet test examples docs functional-test

vet:
	@GO111MODULE=on go vet ./...

test:
	@GO111MODULE=on SERVER_ENDPOINT=play.min.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 MINT_MODE=full go test -race -v ./...

examples:
	@mkdir -p /tmp/examples && for i in $(echo examples/s3/*); do go build -o /tmp/examples/$(basename ${i:0:-3}) ${i}; done

docs:
	@(cd docs; GO111MODULE=on go build validator.go && ./validator -m ../docs/API.md -t checker.go.tpl)

functional-test:
	@GO111MODULE=on SERVER_ENDPOINT=play.min.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 MINT_MODE=full go run functional_tests.go
