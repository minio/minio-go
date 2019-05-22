all: checks

checks:
	@go vet ./...
	@SERVER_ENDPOINT=play.min.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 MINT_MODE=full go test -race -v ./...
	@SERVER_ENDPOINT=play.min.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 MINT_MODE=full go run minio/functional_tests.go
	@mkdir -p /tmp/examples && for i in $(echo examples/s3/*); do go build -o /tmp/examples/$(basename ${i:0:-3}) ${i}; done
	@go build docs/validator.go && ./validator -m docs/API.md -t docs/checker.go.tpl
