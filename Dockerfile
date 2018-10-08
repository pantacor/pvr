FROM golang:alpine as build

WORKDIR /go/src/gitlab.com/pantacor/pvr
COPY . .

RUN apk update; apk add git
RUN version=`git describe --tags` && sed -i "s/NA/$version/" version.go
RUN CGO_ENABLED=0 go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...
RUN CGO_ENABLED=0 GOOS_ARCH=armv6 go get -d -v ./...
RUN CGO_ENABLED=0 GOOS_ARCH=armv6 go install -v ./...

FROM alpine

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /work
COPY --from=build /go/bin/pvr /

ENV USER root

ENTRYPOINT [ "/pvr" ]
