# Build Stage
FROM lacion/docker-alpine:gobuildimage:1.10.3 AS build-stage

LABEL app="build-sugarkube"
LABEL REPO="https://github.com/boosh/sugarkube"

ENV PROJPATH=/go/src/github.com/boosh/sugarkube

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

ADD . /go/src/github.com/boosh/sugarkube
WORKDIR /go/src/github.com/boosh/sugarkube

RUN make build-alpine

# Final Stage
FROM lacion/docker-alpine:latest

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/boosh/sugarkube"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/sugarkube/bin

WORKDIR /opt/sugarkube/bin

COPY --from=build-stage /go/src/github.com/boosh/sugarkube/bin/sugarkube /opt/sugarkube/bin/
RUN chmod +x /opt/sugarkube/bin/sugarkube

CMD /opt/sugarkube/bin/sugarkube
