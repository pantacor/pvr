FROM golang as build

WORKDIR /go/src/gitlab.com/pantacor/pvr
COPY . .

RUN version=`git describe --tags` && sed -i "s/NA/$version/" version.go
RUN go get -d -v ./...
RUN go install -v ./...

FROM debian:stretch-slim

WORKDIR /work
COPY --from=build /go/bin/pvr /

ENTRYPOINT [ "/pvr" ]
