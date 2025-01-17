package main

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/lcpu-club/hpcgame-judger/internal/config"
	"github.com/lcpu-club/hpcgame-judger/internal/manager"
)

func determineKubernetesEndpointFromEnv() string {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return "https://kubernetes.default.svc"
	}

	return "https://" + net.JoinHostPort(host, port)
}

func defaultValue(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func main() {
	conf := &config.ManagerConfig{}
	conf.Listen = flag.String("listen", ":8080", "Listen address")
	conf.Kubernetes = flag.String("kubernetes", determineKubernetesEndpointFromEnv(), "Kubernetes API address")
	conf.KubeSecretPath = flag.String("kube-secret-path", "/var/run/secrets/kubernetes.io/serviceaccount", "Path to the Kubernetes service account token")
	conf.RedisConfig = flag.String("redis-config", "redis://", "Redis configuration")
	conf.SharedVolumePath = flag.String("shared-volume-path", "/data", "Path to shared volume")
	conf.Endpoint = flag.String("endpoint", defaultValue(os.Getenv("ENDPOINT"), "https://hpcgame.pku.edu.cn"), "API endpoint")
	conf.RunnerID = flag.String("runner-id", os.Getenv("RUNNER_ID"), "Runner ID")
	conf.RunnerKey = flag.String("runner-key", os.Getenv("RUNNER_KEY"), "Runner Key")
	conf.RateLimit = flag.Int64("rate-limit", 64, "Rate limit")
	conf.TLSCertFile = flag.String("tls-cert-file", "", "TLS certificate file (empty to disable TLS)")
	conf.TLSKeyFile = flag.String("tls-key-file", "", "TLS key file (empty to disable TLS)")
	conf.TemplatePath = flag.String("template-path", "/templates", "Path to namespace template files")

	flag.Parse()

	s := manager.NewManager(conf)

	err := s.Init()
	if err != nil {
		log.Fatalln(err)
	}

	err = s.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
