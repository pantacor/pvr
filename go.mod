module gitlab.com/pantacor/pvr

go 1.16

require (
	github.com/ChannelMeter/iso8601duration v0.0.0-20150204201828-8da3af7a2a61
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/hcsshim v0.8.7 // indirect
	github.com/asac/json-patch v0.0.0-20201120095033-59358024a068
	github.com/bmatcuk/doublestar v1.3.4
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/channelmeter/iso8601duration v0.0.0-20150204201828-8da3af7a2a61 // indirect
	github.com/containerd/containerd v1.3.3 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.12+incompatible
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/genuinetools/reg v0.16.1
	github.com/gibson042/canonicaljson-go v1.0.3
	github.com/go-jose/go-jose/v3 v3.0.0-rc.1
	github.com/go-resty/resty v1.12.0
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/grandcat/zeroconf v1.0.0
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/justincampbell/bigduration v0.0.0-20160531141349-e45bf03c0666 // indirect
	github.com/justincampbell/timeago v0.0.0-20160528003754-027f40306f1d
	github.com/leekchan/gtf v0.0.0-20190214083521-5fba33c5b00b
	github.com/miekg/dns v1.1.29 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/urfave/cli v1.22.5
	gitlab.com/pantacor/pantahub-base v0.0.0-20220922224408-d1e8099e66c0
	go.mongodb.org/mongo-driver v1.9.0
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/cheggaaa/pb.v1 v1.0.28
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
