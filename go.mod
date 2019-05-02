module github.com/sugarkube/sugarkube

go 1.12

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v0.0.0-20190301161902-9f8fceff796f
	github.com/google/uuid v1.1.1 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/onrik/logrus v0.2.2
	github.com/pelletier/go-toml v1.3.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	golang.org/x/crypto v0.0.0-20190404164418-38d8ce5564a5 // indirect
	gonum.org/v1/gonum v0.0.0-20190416095834-50ee437d7b2f
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

// using our custom fork
replace gopkg.in/yaml.v2 v2.2.2 => github.com/sugarkube/yaml v2.1.0+incompatible
