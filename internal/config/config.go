package config

type ManagerConfig struct {
	Listen     *string
	Kubernetes *string

	Endpoint  *string
	RunnerID  *string
	RunnerKey *string
	RateLimit *int64

	RedisConfig      *string
	SharedVolumePath *string

	KubeSecretPath *string

	TLSCertFile *string
	TLSKeyFile  *string

	TemplatePath *string
}
