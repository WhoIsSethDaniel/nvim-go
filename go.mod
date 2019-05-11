module github.com/zchee/nvim-go

require (
	cloud.google.com/go v0.38.0
	contrib.go.opencensus.io/exporter/stackdriver v0.11.1-0.20190509233537-79f9672ef5ad
	github.com/DataDog/datadog-go v2.2.1-0.20190425163447-40bafcb5f6c1+incompatible // indirect
	github.com/DataDog/opencensus-go-exporter-datadog v0.0.0-20190503082300-0f32ad59ab08
	github.com/aws/aws-sdk-go v1.19.28 // indirect
	github.com/cweill/gotests v1.5.3-0.20181029041911-276664f3b507
	github.com/davecgh/go-spew v1.1.1
	github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0
	github.com/fatih/color v1.6.0 // indirect
	github.com/google/go-cmp v0.3.0
	github.com/google/pprof v0.0.0-20190404155422-f8f10df84213 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/haya14busa/errorformat v0.0.0-20180607161917-689b7d67b7a8
	github.com/hokaccha/go-prettyjson v0.0.0-20180920040306-f579f869bbfe
	github.com/kelseyhightower/envconfig v1.3.1-0.20180517194557-dd1402a4d99d
	github.com/mattn/go-colorable v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/motemen/go-astmanip v0.0.0-20160104081417-d6ad31f02153
	github.com/neovim/go-client v0.0.0-20190511104716-eaaca6f62369
	github.com/peterh/liner v1.1.0 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pkg/profile v1.2.1 // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/zchee/color v1.7.0
	github.com/zchee/go-xdgbasedir v1.0.3
	go.opencensus.io v0.21.0
	go.uber.org/atomic v1.3.3-0.20190226011305-5328d69c76a9 // indirect
	go.uber.org/multierr v1.1.1-0.20180122172545-ddea229ff1df
	go.uber.org/zap v1.9.2-0.20190327195448-badef736563f
	golang.org/x/arch v0.0.0-20170711125641-f40095975f84 // indirect
	golang.org/x/debug v0.0.0-20160621010512-fb508927b491 // indirect
	golang.org/x/exp/errors v0.0.0-20190221220918-438050ddec5e
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/net v0.0.0-20190509222800-a4d6f7feada5 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190509141414-a5b02f93d862
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/tools v0.0.0-20190312170243-e65039ee4138
	google.golang.org/api v0.5.0 // indirect
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20190508193815-b515fa19cec8 // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.13.0 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20190409140830-cdc409dda467
)

replace (
	github.com/derekparker/delve v1.2.0 => github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0
	github.com/fatih/color v1.7.0 => github.com/zchee/color v1.7.1-0.20190331162438-438c6d2abc51
	github.com/go-delve/delve v1.2.0 => github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0
	github.com/go-resty/resty v1.9.0 => gopkg.in/resty.v1 v1.9.0
	github.com/googleapis/gax-go/v2 v2.0.0 => github.com/googleapis/gax-go/v2 v2.0.3
	golang.org/x/tools v0.0.0-20181030000716-a0a13e073c7b => golang.org/x/tools v0.0.0-20190410211219-2538eef75904
)
