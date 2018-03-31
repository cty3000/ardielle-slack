FROM ubuntu:latest
RUN mkdir -p /opt/slackd/bin
ADD ./go/bin/linux_amd64/slackd /opt/slackd/bin/slackd
EXPOSE 4080
CMD /opt/slackd/bin/slackd
