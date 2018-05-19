export GOPATH=$(PWD)/go
RDL ?= $(GOPATH)/bin/rdl

all: go/bin/slackd

go/bin/slackd: go/src/slackd go/src/slack go/src/github.com/dimfeld/httptreemux
	go install slackd
	GOOS=linux go install slackd

go/bin/slack-cli: go/src/slack
	#go install slack-cli
	#GOOS=linux go install slack-cli

go/src/github.com/dimfeld/httptreemux:
	go get github.com/dimfeld/httptreemux

go/src/slack: rdl/slackd.rdl rdl/slack-cli.rdl $(RDL)
	mkdir -p go/src/slack
	$(RDL) -ps generate -t -o go/src/slack go-model rdl/slackd.rdl
	$(RDL) -ps generate -t -o go/src/slack go-server rdl/slackd.rdl
	$(RDL) -ps generate -t -o go/src/slack go-client rdl/slack-cli.rdl

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
	rm -rf go/bin go/pkg go/src go/slack
