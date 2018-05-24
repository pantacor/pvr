FROM golang:alpine as build

WORKDIR /go/src/gitlab.com/pantacor/pvr
COPY . .

RUN apk update; apk add git
RUN version=`git describe --tags` && sed -i "s/NA/$version/" version.go
RUN go get -d -v ./...
RUN go install -v ./...

FROM alpine

WORKDIR /work
COPY --from=build /go/bin/pvr /

ENTRYPOINT [ "/pvr" ]
