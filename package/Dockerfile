FROM ubuntu:16.04
RUN mv /etc/apt/sources.list /etc/apt/sources.list.bk
ADD http://dev-tool.oss-cn-shenzhen.aliyuncs.com/sources.list /etc/apt/sources.list
ADD http://dev-tool.oss-cn-shenzhen.aliyuncs.com/git/tools.tar.gz /tmp/tools.tar.gz
ADD zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
RUN apt-get update && apt-get install -y curl ca-certificates git && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /data/rancher-ci/templates /data/rancher-ci/jenkins /data/rancher-ci/workspace
ENV JENKINS_HOME /data/rancher-ci/workspace
COPY pipeline /usr/bin/
CMD ["pipeline"]
