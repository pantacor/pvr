FROM golang:alpine as build

WORKDIR /go/src/gitlab.com/pantacor/pvr
COPY . .

RUN apk update; apk add git
RUN version=`git describe --tags` && sed -i "s/NA/$version/" version.go

# build amd64 linux static
RUN CGO_ENABLED=0 GOOS=linux GOOS_ARCH=amd64  go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOOS_ARCH=amd64 go install -v ./...

# build armv6 linux static
RUN CGO_ENABLED=0 GOOS=linux GOOS_ARCH=armv6 go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOOS_ARCH=armv6 go install -v ./...

# build windows i386 static
RUN CGO_ENABLED=0 GOOS=windows GOOS_ARCH=i386 go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=windows GOOS_ARCH=i386 go install -v ./...

# build windows amd64 static
RUN CGO_ENABLED=0 GOOS=windows GOOS_ARCH=amd64 go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=windows GOOS_ARCH=amd64 go install -v ./...

FROM alpine

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /work
COPY --from=build /go/bin /pvr

ENV USER root

ENTRYPOINT [ "/bin/sh" ]
