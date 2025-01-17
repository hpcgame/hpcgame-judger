package manager

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const nsPrefix = "j-"

type RunningConfig struct {
	JobTemplate *batchv1.Job           `json:"jobTemplate"`
	Variables   map[string]interface{} `json:"variables"`
}

func (s *JudgeSession) run() error {
	defer s.runningCleanup()

	err := s.ensureNamespacePresence()
	if err != nil {
		return err
	}

	err = s.ensureJobPresence()
	if err != nil {
		return err
	}

	return s.watchJob()
}

func (s *JudgeSession) GetNamespaceName() string {
	return nsPrefix + s.soln.TaskId
}

func (s *JudgeSession) ensureNamespacePresence() error {
	nsName := s.GetNamespaceName()

	_, err := s.m.kc.Client().CoreV1().Namespaces().Get(context.TODO(), nsName, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	err = s.createNamespace()
	if err != nil {
		return err
	}

	return nil
}

func (s *JudgeSession) createNamespace() error {
	nsName := s.GetNamespaceName()

	values := &struct {
		Namespace string
		Variables map[string]interface{}
	}{
		Namespace: nsName,
		Variables: s.rc.Variables,
	}

	for _, tmpl := range s.m.tmpls {
		buf := bytes.NewBuffer(nil)
		err := tmpl.Execute(buf, values)
		if err != nil {
			return err
		}

		err = s.m.kc.Create(context.TODO(), buf.String(), false)

		if err != nil {
			return err
		}
	}

	log.Println("Created namespace", nsName)

	return nil
}

const deleteNamespaceGracePeriods = 5

func (s *JudgeSession) deleteNamespace() error {
	err := s.m.kc.DeleteNamespace(context.TODO(), s.GetNamespaceName(), deleteNamespaceGracePeriods)
	if err != nil {
		return err
	}

	log.Println("Deleted namespace", s.GetNamespaceName())
	return err
}

func (s *JudgeSession) GetJobName() string {
	// HARDCODED NAME
	return "judge"
}

func (s *JudgeSession) ensureJobPresence() error {
	jobName := s.GetJobName()

	_, err := s.m.kc.Client().BatchV1().Jobs(s.GetNamespaceName()).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	err = s.createJob()
	if err != nil {
		return err
	}

	log.Println("Created job", s.GetNamespaceName(), jobName)

	return nil
}

func (s *JudgeSession) createJob() error {
	if s.rc.JobTemplate == nil {
		return fmt.Errorf("job template is nil")
	}

	job := s.rc.JobTemplate.DeepCopy()
	job.Namespace = s.GetNamespaceName()
	job.Name = s.GetJobName()

	// Insert download environment variables
	const varName = "SOLUTION_URL"
	for k := range job.Spec.Template.Spec.Containers {
		for _, env := range job.Spec.Template.Spec.Containers[k].Env {
			if env.Name == varName {
				// Unset if exists
				job.Spec.Template.Spec.Containers[k].Env = append(job.Spec.Template.Spec.Containers[k].Env[:k], job.Spec.Template.Spec.Containers[k].Env[k+1:]...)
				break
			}
		}

		job.Spec.Template.Spec.Containers[k].Env = append(job.Spec.Template.Spec.Containers[k].Env, corev1.EnvVar{
			Name:  varName,
			Value: s.soln.SolutionDataUrl,
		})
	}

	_, err := s.m.kc.Client().BatchV1().Jobs(s.GetNamespaceName()).Create(context.TODO(), job, metav1.CreateOptions{})
	return err
}

func (s *JudgeSession) getPodLogsOfJob(follow bool, since *time.Time) (io.ReadCloser, error) {
	jobName := s.GetJobName()

	pods, err := s.m.kc.Client().CoreV1().Pods(s.GetNamespaceName()).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, nil
	}

	podName := pods.Items[0].Name

	opts := &corev1.PodLogOptions{
		Follow: follow,
	}
	if since != nil {
		opts.SinceTime = &metav1.Time{Time: *since}
	}
	logReq := s.m.kc.Client().CoreV1().Pods(s.GetNamespaceName()).GetLogs(podName, opts)
	reader, err := logReq.Stream(context.TODO())

	return reader, err
}

func (s *JudgeSession) processedTimestampKey() string {
	return fmt.Sprintf("judge:processed:%s", s.id)
}

func (s *JudgeSession) updateProcessedTimestamp(t *time.Time) error {
	return s.m.r.Client.Set(context.TODO(), s.processedTimestampKey(), t.Format(time.RFC3339), 0).Err()
}

func (s *JudgeSession) getProcessedTimestamp() (*time.Time, error) {
	t, err := s.m.r.Client.Get(context.TODO(), s.processedTimestampKey()).Result()

	// If not exist, return nil, nil
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func (s *JudgeSession) deleteProcessedTimestamp() error {
	return s.m.r.Client.Del(context.TODO(), s.processedTimestampKey()).Err()
}

func (s *JudgeSession) runningCleanup() {
	err := s.deleteProcessedTimestamp()
	if err != nil {
		log.Println("Failed to delete processed timestamp:", err)
	}
	err = s.deleteNamespace()
	if err != nil {
		log.Println("Failed to delete namespace:", err)
	}
}

const watchJobTimeout = 10 * time.Minute

func (s *JudgeSession) watchJobTillReady() error {
	jobName := s.GetJobName()

	watcher, err := s.m.kc.Client().BatchV1().Jobs(s.GetNamespaceName()).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", jobName),
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	// Wait for the job to start running
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watcher channel closed unexpectedly")
			}

			job, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			// Check if the job has started running
			if job.Status.Active > 0 {
				if job.Status.Ready != nil && *job.Status.Ready > 0 {
					return nil
				}
			}

			if event.Type != watch.Added && event.Type != watch.Modified {
				return fmt.Errorf("job not running: %s", job.Status.String())
			}
		case <-time.After(watchJobTimeout):
			return fmt.Errorf("timed out waiting for job to start")
		}
	}
}

func (s *JudgeSession) watchJob() error {
	err := s.watchJobTillReady()
	if err != nil {
		return err
	}

	log.Println("Job started running", s.GetNamespaceName())

	// Get the timestamp before starting the log pulling loop
	since, err := s.getProcessedTimestamp()
	if err != nil {
		return err
	}

	// Start the log pulling loop
	return s.pullJobLogs(since)
}

func (s *JudgeSession) pullJobLogs(since *time.Time) error {
	reader, err := s.getPodLogsOfJob(true, since)
	if err != nil {
		return err
	}
	defer reader.Close()

	buf := bufio.NewReader(reader)
	for {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		lineStr := string(line)
		if lineStr == "" {
			continue
		}

		// Process each line as an action
		// TODO: real processing logic
		log.Printf("Action: %s", lineStr)
		err = s.processMessage(lineStr)
		if err != nil {
			return err
		}

		// Update the processed timestamp
		now := time.Now()
		if err := s.updateProcessedTimestamp(&now); err != nil {
			return err
		}
	}

	// MUST complete the job, otherwise maybe not completed
	s.aoi.Complete(context.TODO())

	return nil
}
