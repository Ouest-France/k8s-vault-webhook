module github.com/Ouest-France/k8s-vault-webhook

go 1.13

require (
	github.com/Masterminds/sprig/v3 v3.2.0
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/render v1.0.1
	github.com/google/uuid v1.1.5 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	k8s.io/api v0.0.0-20191010143144-fbf594f18f80
	k8s.io/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
)
