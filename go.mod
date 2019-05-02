module github.com/sugarkube/sugarkube

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/google/uuid v1.1.1 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/onrik/logrus v0.2.2
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	gonum.org/v1/gonum v0.0.0-20190430210020-9827ae2933ff
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

// using our custom fork
replace gopkg.in/yaml.v2 => github.com/sugarkube/yaml v0.0.0-20190303195351-8c2d5c55e5e0
