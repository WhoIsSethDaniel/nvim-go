module github.com/zchee/nvim-go

require (
	cloud.google.com/go v0.37.4
	contrib.go.opencensus.io/exporter/stackdriver v0.10.2
	git.apache.org/thrift.git v0.0.0-20180902110319-2566ecd5d999 // indirect
	github.com/DataDog/datadog-go v0.0.0-20190315133836-a5d50a065561 // indirect
	github.com/DataDog/opencensus-go-exporter-datadog v0.0.0-20190314110122-1e6ba4554ec1
	github.com/cweill/gotests v1.5.3-0.20181029041911-276664f3b507
	github.com/davecgh/go-spew v1.1.1
	github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0
	github.com/fatih/color v1.6.0 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/go-cmp v0.2.1-0.20190312032427-6f77996f0c42
	github.com/haya14busa/errorformat v0.0.0-20180607161917-689b7d67b7a8
	github.com/hokaccha/go-prettyjson v0.0.0-20180920040306-f579f869bbfe
	github.com/kelseyhightower/envconfig v1.3.1-0.20180517194557-dd1402a4d99d
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/motemen/go-astmanip v0.0.0-20160104081417-d6ad31f02153
	github.com/neovim/go-client v0.0.0-20190408193136-de4d01378b14
	github.com/peterh/liner v1.1.0 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pkg/profile v1.2.1 // indirect
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/zchee/color v1.7.0
	github.com/zchee/go-xdgbasedir v1.0.3
	go.opencensus.io v0.20.2
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.1-0.20180122172545-ddea229ff1df
	go.uber.org/zap v1.9.2-0.20190327195448-badef736563f
	golang.org/x/arch v0.0.0-20170711125641-f40095975f84 // indirect
	golang.org/x/build v0.0.0-20190111050920-041ab4dc3f9d // indirect
	golang.org/x/debug v0.0.0-20160621010512-fb508927b491 // indirect
	golang.org/x/exp/errors v0.0.0-20190221220918-438050ddec5e
	golang.org/x/lint v0.0.0-20190301231843-5614ed5bae6f
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a
	golang.org/x/tools v0.0.0-20190312170243-e65039ee4138
	gopkg.in/DataDog/dd-trace-go.v1 v1.10.0 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/googleapis/gax-go/v2 v2.0.0 => github.com/googleapis/gax-go/v2 v2.0.3

replace golang.org/x/tools v0.0.0-20181030000716-a0a13e073c7b => golang.org/x/tools v0.0.0-20190219185102-9394956cfdc5

replace github.com/go-delve/delve v1.2.0 => github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0

replace github.com/derekparker/delve v1.2.0 => github.com/derekparker/delve v0.12.3-0.20170419170936-92dad944d7e0

replace github.com/fatih/color v1.7.0 => github.com/zchee/color v1.7.1-0.20190331162438-438c6d2abc51
