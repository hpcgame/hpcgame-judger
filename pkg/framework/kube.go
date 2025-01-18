package framework

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/lcpu-club/hpcgame-judger/internal/kube"
	"github.com/lcpu-club/hpcgame-judger/internal/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

func NS() string {
	return Kube().Namespace()
}

func WaitJobAndGetPods(job string) ([]string, error) {
	c := Kube().Client()

	watcher, err := c.BatchV1().Jobs(NS()).Watch(BgCtx(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job),
	})
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil, fmt.Errorf("watcher channel closed unexpectedly")
			}

			job, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			if event.Type != watch.Added && event.Type != watch.Modified {
				return nil, fmt.Errorf("job not running: %s", job.Status.String())
			}

			if job.Status.Active > 0 {
				if (job.Status.Ready != nil && *job.Status.Ready > 0) ||
					job.Status.Succeeded > 0 || job.Status.Failed > 0 {
					pods, err := c.CoreV1().Pods(NS()).List(BgCtx(), metav1.ListOptions{
						LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
					})
					if err != nil {
						return nil, err
					}

					var podNames []string
					for _, pod := range pods.Items {
						podNames = append(podNames, pod.Name)
					}

					return podNames, nil
				}
			}
		case <-time.After(5 * time.Minute):
			return nil, fmt.Errorf("timed out waiting for job to start")
		}
	}
}

func PodLogs(pod string) (io.ReadCloser, error) {
	return Kube().Client().CoreV1().Pods(NS()).GetLogs(pod, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(BgCtx())
}

func JobSuccessOrNot(job string) (bool, error) {
	c := Kube().Client()

	jobObj, err := c.BatchV1().Jobs(NS()).Get(BgCtx(), job, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	return jobObj.Status.Succeeded > 0, nil
}
