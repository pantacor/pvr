module gitlab.com/pantacor/pvr

go 1.16

require (
	github.com/ChannelMeter/iso8601duration v0.0.0-20150204201828-8da3af7a2a61
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.8.7 // indirect
	github.com/asac/json-patch v0.0.0-20201120095033-59358024a068
	github.com/bmatcuk/doublestar v1.3.4
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/containerd/containerd v1.3.3 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.12+incompatible
	github.com/fatih/color v1.7.0 // indirect
	github.com/fluent/fluent-logger-golang v1.5.0 // indirect
	github.com/genuinetools/reg v0.16.1
	github.com/gibson042/canonicaljson-go v1.0.3
	github.com/go-jose/go-jose/v3 v3.0.0-rc.1
	github.com/go-resty/resty v1.12.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/grandcat/zeroconf v1.0.0
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/justincampbell/bigduration v0.0.0-20160531141349-e45bf03c0666 // indirect
	github.com/justincampbell/timeago v0.0.0-20160528003754-027f40306f1d
	github.com/klauspost/compress v1.10.4 // indirect
	github.com/leekchan/gtf v0.0.0-20190214083521-5fba33c5b00b
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/miekg/dns v1.1.29 // indirect
	github.com/miolini/datacounter v1.0.2 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/tinylib/msgp v1.1.2 // indirect
	github.com/urfave/cli v1.22.4
	gitlab.com/pantacor/pantahub-base v0.0.0-20200517092730-d03429894e0c
	go.mongodb.org/mongo-driver v1.5.1
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
	google.golang.org/genproto v0.0.0-20200413115906-b5235f65be36 // indirect
	google.golang.org/grpc v1.28.1 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/olivere/elastic.v5 v5.0.85 // indirect
)

replace github.com/ant0ine/go-json-rest => github.com/asac/go-json-rest v3.3.3-0.20191004094541-40429adaafcb+incompatible

replace github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190723064612-a9dc697fd2a5

exclude github.com/Sirupsen/logrus v1.4.0

exclude github.com/Sirupsen/logrus v1.3.0

exclude github.com/Sirupsen/logrus v1.2.0

exclude github.com/Sirupsen/logrus v1.1.1

exclude github.com/Sirupsen/logrus v1.1.0

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => github.com/golang/lint v0.0.0-20190301231843-5614ed5bae6f

replace github.com/go-jose/go-jose/v3 => github.com/asac/go-jose/v3 v3.0.0-20210726220436-d8aa79561ce4
