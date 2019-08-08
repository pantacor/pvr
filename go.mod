module gitlab.com/pantacor/pvr

go 1.12

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible
	github.com/Sirupsen/logrus v1.0.6 // indirect
	github.com/asac/json-patch v0.0.0-20170214153119-c7d7c4ba959b
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/containerd/containerd v1.2.7 // indirect
	github.com/docker/cli v0.0.0-20190723233319-62f123fbd2ec
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/docker-ce v0.0.0-20190724010320-53720a99f3c5
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/genuinetools/reg v0.16.1-0.20190610165748-e29a4fdc2e43
	github.com/go-resty/resty v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.1 // indirect
	github.com/grandcat/zeroconf v0.0.0-20190424104450-85eadb44205c
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/justincampbell/timeago v0.0.0-20160528003754-027f40306f1d
	github.com/leekchan/gtf v0.0.0-20190214083521-5fba33c5b00b
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/miekg/dns v1.1.15 // indirect
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/urfave/cli v1.20.0
	github.com/vbatts/tar-split v0.11.1 // indirect
	gitlab.com/pantacor/pantahub-base v0.0.0-20190724204618-9041e257b8c8
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	golang.org/x/tools v0.0.0-20190723021737-8bb11ff117ca // indirect
	google.golang.org/grpc v1.20.1
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/resty.v1 v1.12.0
)

replace github.com/ant0ine/go-json-rest => github.com/asac/go-json-rest v3.3.3-0.20181121222456-cab770813df3+incompatible

replace github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190723064612-a9dc697fd2a5

exclude github.com/Sirupsen/logrus v1.4.0

exclude github.com/Sirupsen/logrus v1.3.0

exclude github.com/Sirupsen/logrus v1.2.0

exclude github.com/Sirupsen/logrus v1.1.1

exclude github.com/Sirupsen/logrus v1.1.0

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => github.com/golang/lint v0.0.0-20190409202823-5614ed5bae6fb75893070bdc0996a68765fdd275
