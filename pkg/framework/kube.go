package framework

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lcpu-club/hpcgame-judger/internal/kube"
	"github.com/lcpu-club/hpcgame-judger/internal/utils"
	"github.com/lcpu-club/hpcgame-judger/pkg/judgerproto"
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

func calcReadyAndFinishedPods(job batchv1.JobStatus) int {
	ready := 0
	if job.Ready != nil {
		ready = int(*job.Ready)
	}
	// active := int(job.Active)
	finished := int(job.Succeeded + job.Failed)
	terminating := 0
	if job.Terminating != nil {
		terminating = int(*job.Terminating)
	}
	return ready + finished + terminating
}

func WaitJobAndGetPods(job string, requiredPods int) ([]string, error) {
	c := Kube().Client()

	watcher, err := c.BatchV1().Jobs(NS()).Watch(BgCtx(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job),
	})
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	state, err := c.BatchV1().Jobs(NS()).Get(BgCtx(), job, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if calcReadyAndFinishedPods(state.Status) >= requiredPods {
		pods, err := c.CoreV1().Pods(NS()).List(BgCtx(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job),
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

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			{
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

				if calcReadyAndFinishedPods(job.Status) >= requiredPods {
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
		case <-time.After(120 * time.Minute):
			return nil, fmt.Errorf("timed out waiting for job %s to start", job)
		}
	}
}

func PodLogs(pod string) (io.ReadCloser, error) {
	// _, err := WaitTill(func() (*corev1.Pod, error) {
	// 	return Kube().Client().CoreV1().Pods(NS()).Get(BgCtx(), pod, metav1.GetOptions{})
	// }, func(p *corev1.Pod) bool {
	// 	for _, ctr := range p.Status.ContainerStatuses {
	// 		if ctr.Started == nil || !*ctr.Started {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// }, 48, 200*time.Millisecond)
	// if err != nil {
	// 	return nil, err
	// }
	backOffStart := 200 * time.Millisecond
	for range 24 {
		logger, err := Kube().Client().CoreV1().Pods(NS()).GetLogs(pod, &corev1.PodLogOptions{
			Follow: true,
		}).Stream(BgCtx())
		if strings.Contains(err.Error(), "ContainerCreating") {
			time.Sleep(backOffStart)
			backOffStart = expCoolDown(backOffStart, 8*time.Second)
			continue
		}
		if err != nil {
			return nil, err
		}
		return logger, nil
	}
	return nil, fmt.Errorf("backoff limit reached waiting for pod %s to quit ContainerCreating", pod)
}

func JobSuccessOrNot(job string) (bool, error) {
	c := Kube().Client()

	jobObj, err := c.BatchV1().Jobs(NS()).Get(BgCtx(), job, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	judgerproto.NewLogMessage(fmt.Sprintf("Job status: %#v", jobObj.Status)).Print()

	return jobObj.Status.Failed == 0, nil
}

func PodSuccessOrNot(pod string) (bool, error) {
	c := Kube().Client()

	podObj, err := WaitTill(func() (*corev1.Pod, error) {
		return c.CoreV1().Pods(NS()).Get(BgCtx(), pod, metav1.GetOptions{})
	}, func(p *corev1.Pod) bool {
		return p.Status.Phase != corev1.PodRunning
	}, 48, 350*time.Millisecond)
	if err != nil {
		return false, err
	}

	return podObj.Status.Phase == corev1.PodSucceeded, nil
}

func WaitPods(pods []string) (output []string, success bool, err error) {
	for _, pod := range pods {
		logs, err := PodLogs(pod)
		if err != nil {
			return nil, false, err
		}
		defer logs.Close()

		buf := new(strings.Builder)
		_, err = io.Copy(buf, logs)
		if err != nil {
			return nil, false, err
		}

		output = append(output, buf.String())
	}

	for _, pod := range pods {
		success, err = PodSuccessOrNot(pod)
		if err != nil {
			return nil, false, err
		}

		if !success {
			return output, false, nil
		}
	}

	return output, true, err
}

func WaitJobFinish(job string, requiredPods int) (output []string, success bool, err error) {
	pods, err := WaitJobAndGetPods(job, requiredPods)
	if err != nil {
		return nil, false, err
	}

	return WaitPods(pods)
}
