module gitlab.com/pantacor/pvr

go 1.12

require (
	cloud.google.com/go v0.38.0 // indirect
	github.com/ChannelMeter/iso8601duration v0.0.0-20150204201828-8da3af7a2a61
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.8.7 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/asac/json-patch v0.0.0-20201120095033-59358024a068
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/containerd/containerd v1.3.3 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/coreos/clair v2.0.8+incompatible // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/docker-ce v0.0.0-20190724010320-53720a99f3c5 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/fluent/fluent-logger-golang v1.5.0 // indirect
	github.com/genuinetools/reg v0.16.1
	github.com/go-resty/resty v1.12.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/lint v0.0.0-20190409202823-959b441ac422 // indirect
	github.com/golang/mock v1.3.0 // indirect
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/pprof v0.0.0-20190502144155-8358a9778bd1 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/context v1.1.1 // indirect
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
	github.com/rogpeppe/fastuuid v1.0.0 // indirect
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/tinylib/msgp v1.1.2 // indirect
	github.com/urfave/cli v1.22.4
	gitlab.com/pantacor/pantahub-base v0.0.0-20200517092730-d03429894e0c
	go.mongodb.org/mongo-driver v1.3.2
	golang.org/x/crypto v0.0.0-20200406173513-056763e48d71
	golang.org/x/exp v0.0.0-20190429183610-475c5042d3f1 // indirect
	golang.org/x/image v0.0.0-20190501045829-6d32002ffd75 // indirect
	golang.org/x/mobile v0.0.0-20190415191353-3e0bab5405d6 // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a // indirect
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4 // indirect
	google.golang.org/genproto v0.0.0-20200413115906-b5235f65be36 // indirect
	google.golang.org/grpc v1.28.1 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/olivere/elastic.v5 v5.0.85 // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/square/go-jose.v2 v2.5.0 // indirect
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
