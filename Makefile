all: checks

checks:
	@go get -u github.com/go-ini/ini/...
	@go get -u github.com/mitchellh/go-homedir/...
	@go get -u github.com/cheggaaa/pb/...
	@go get -u github.com/sirupsen/logrus/...
	@go get -u github.com/dustin/go-humanize/...
	@go vet ./...
	@SERVER_ENDPOINT=play.minio.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 go test -race -v ./...
	@SERVER_ENDPOINT=play.minio.io:9000 ACCESS_KEY=Q3AM3UQ867SPQQA43P2F SECRET_KEY=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG ENABLE_HTTPS=1 go run functional_tests.go
	@mkdir -p /tmp/examples && for i in $(echo examples/s3/*); do go build -o /tmp/examples/$(basename ${i:0:-3}) ${i}; done
	@go get -u github.com/a8m/mark/...
	@go get -u github.com/minio/cli/...
	@go get -u golang.org/x/tools/cmd/goimports
	@go get -u github.com/gernest/wow/...
	@go build docs/validator.go && ./validator -m docs/API.md -t docs/checker.go.tpl
