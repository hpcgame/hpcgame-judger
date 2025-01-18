package framework

import (
	"net"
	"os"

	"github.com/lcpu-club/hpcgame-judger/internal/kube"
	"github.com/lcpu-club/hpcgame-judger/internal/utils"
)

var kubeClient *kube.Client = nil

func determineKubernetesEndpointFromEnv() string {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return "https://kubernetes.default.svc"
	}

	return "https://" + net.JoinHostPort(host, port)
}

func getSecretManager() *utils.SecretManager {
	return utils.NewSecretManager("/var/run/secrets/kubernetes.io/serviceaccount")
}

func initKube() {
	kubeClient = Must(kube.NewClient(determineKubernetesEndpointFromEnv(), getSecretManager()))
}

func Kube() *kube.Client {
	if kubeClient == nil {
		initKube()
	}
	return kubeClient
}
