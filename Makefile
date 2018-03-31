export GOPATH=$(PWD)/go
RDL ?= $(GOPATH)/bin/rdl

all: go/bin/slackd

go/bin/slackd: keys go/src/slackd go/src/slack go/src/github.com/dimfeld/httptreemux
	go install slackd
	GOOS=linux go install slackd

keys:
	rm -rf keys certs
	go run ca/gencerts.go

keys/client.p12: keys/client.key
	openssl pkcs12 -password pass:example -export -in ./certs/client.cert -inkey ./keys/client.key -out ./keys/client.p12

test-curl: keys/client.p12
	curl --cacert certs/ca.cert -E ./keys/client.p12:example https://localhost:4443/example/v1/slack/$(USER)  -X DELETE

go/src/github.com/dimfeld/httptreemux:
	go get github.com/dimfeld/httptreemux

go/src/slack: rdl/slack.rdl $(RDL)
	mkdir -p go/src/slack
	$(RDL) -ps generate -t -o go/src/slack go-model rdl/slack.rdl
	$(RDL) -ps generate -t -o go/src/slack go-server rdl/slack.rdl
	$(RDL) -ps generate -t -o go/src/slack go-client rdl/slack.rdl

go/src/slackd:
	mkdir -p go/src
	(cd go/src; ln -s ../slackd)

$(RDL):
	go get github.com/ardielle/ardielle-tools/...

bin/$(NAME): generated src/slackd/main.go
	go install $(NAME)

src/slackd/main.go:
	(cd src; ln -s .. slackd)

clean::
	rm -rf go/bin go/pkg go/src keys certs go/slack
