# Build Stage
FROM lacion/docker-alpine:gobuildimage:1.10.3 AS build-stage

LABEL app="build-sugarkube"
LABEL REPO="https://github.com/sugarkube/sugarkube"

ENV PROJPATH=/go/src/github.com/sugarkube/sugarkube

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

ADD . /go/src/github.com/sugarkube/sugarkube
WORKDIR /go/src/github.com/sugarkube/sugarkube

RUN make build-alpine

# Final Stage
FROM lacion/docker-alpine:latest

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/sugarkube/sugarkube"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/sugarkube/bin

WORKDIR /opt/sugarkube/bin

COPY --from=build-stage /go/src/github.com/sugarkube/sugarkube/bin/sugarkube /opt/sugarkube/bin/
RUN chmod +x /opt/sugarkube/bin/sugarkube

CMD /opt/sugarkube/bin/sugarkube
