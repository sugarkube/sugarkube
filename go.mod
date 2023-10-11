module github.com/sugarkube/sugarkube

go 1.12

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mattn/go-shellwords v1.0.6
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db
	github.com/onrik/logrus v0.2.2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.1
	github.com/skratchdot/open-golang v0.0.0-20190402232053-79abb63cd66e
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.17.0 // indirect
	gonum.org/v1/gonum v0.0.0-20190430210020-9827ae2933ff
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/client-go v0.20.0-alpha.2
)

// using our custom fork
replace gopkg.in/yaml.v2 => github.com/sugarkube/yaml v0.0.0-20190303195351-8c2d5c55e5e0
