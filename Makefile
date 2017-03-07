default: .uptodate

ORG          = paulbellamy
REPO         = kubewatch
DOCKER_REPO  = ${ORG}/${REPO}
VERSION     ?= latest

# Building and testing

.uptodate: kubewatch
	@docker build -t ${DOCKER_REPO}:${VERSION} .

kubewatch: $(shell find . -name '*.go')
	GOOS=linux GOARCH=amd64 go build -o $@ .

clean:
	- rm kubewatch .uptodate
