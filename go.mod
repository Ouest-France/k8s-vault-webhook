module github.com/Ouest-France/k8s-vault-webhook

go 1.13

require (
	github.com/Masterminds/sprig/v3 v3.0.0
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/render v1.0.1
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	k8s.io/api v0.0.0-20191010143144-fbf594f18f80
	k8s.io/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
)
