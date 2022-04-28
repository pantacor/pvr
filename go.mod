module gitlab.com/pantacor/pvr/v3

go 1.16

replace github.com/ant0ine/go-json-rest => github.com/asac/go-json-rest v3.3.3-0.20191004094541-40429adaafcb+incompatible

replace github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190723064612-a9dc697fd2a5

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => github.com/golang/lint v0.0.0-20190301231843-5614ed5bae6f

replace github.com/go-jose/go-jose/v3 => github.com/asac/go-jose/v3 v3.0.0-20210726220436-d8aa79561ce4

require (
	github.com/ChannelMeter/iso8601duration v0.0.0-20150204201828-8da3af7a2a61
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/asac/json-patch v0.0.0-20201120095033-59358024a068
	github.com/bmatcuk/doublestar v1.3.4
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.12+incompatible
	github.com/genuinetools/reg v0.16.1
	github.com/gibson042/canonicaljson-go v1.0.3
	github.com/go-jose/go-jose/v3 v3.0.0-rc.1
	github.com/go-resty/resty v1.12.0
	github.com/grandcat/zeroconf v1.0.0
	github.com/justincampbell/timeago v0.0.0-20160528003754-027f40306f1d
	github.com/leekchan/gtf v0.0.0-20190214083521-5fba33c5b00b
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/go-digest v1.0.0
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/urfave/cli v1.22.8
	gitlab.com/pantacor/pantahub-base v0.0.0-20220408222804-5c7f3643cdef
	gitlab.com/pantacor/pvr v0.0.0-20220419081307-3b01432b96af
	go.mongodb.org/mongo-driver v1.9.0
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171
	gopkg.in/cheggaaa/pb.v1 v1.0.28
)
