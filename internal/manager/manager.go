package manager

import (
	"errors"
	"log"
	"os"
	"text/template"

	"github.com/lcpu-club/hpcgame-judger/internal/config"
	"github.com/lcpu-club/hpcgame-judger/internal/kube"
	"github.com/lcpu-club/hpcgame-judger/internal/utils"
	"github.com/lcpu-club/hpcgame-judger/pkg/aoiclient"
)

type Manager struct {
	conf *config.ManagerConfig
	sm   *utils.SecretManager
	kc   *kube.Client
	aoi  *aoiclient.Client
	r    *Redis
	rl   *RateLimiter

	managerID string

	tmpls []*template.Template
}

func NewManager(conf *config.ManagerConfig) *Manager {
	return &Manager{
		conf: conf,
		sm:   utils.NewSecretManager(*conf.KubeSecretPath),
	}
}

func (m *Manager) genID() {
	m.managerID = ""

	// First acquire hostname
	hostname, err := os.Hostname()
	if err == nil {
		m.managerID = hostname + "-"
	}

	// Then use a random string
	m.managerID += utils.GenerateRandomString(6, "")

	log.Println("Using manager ID:", m.managerID)
}

func (m *Manager) Init() error {
	kc, err := kube.NewClient(*m.conf.Kubernetes, m.sm)
	if err != nil {
		return err
	}
	m.kc = kc

	aoi := aoiclient.New(*m.conf.Endpoint)
	if *m.conf.RunnerID != "" || *m.conf.RunnerKey != "" {
		aoi.Authenticate(*m.conf.RunnerID, *m.conf.RunnerKey)
	} else {
		return errors.New("runner ID and key must be provided")
	}
	m.aoi = aoi

	r, err := NewRedis(*m.conf.RedisConfig)
	if err != nil {
		return err
	}
	m.r = r

	m.genID()

	err = m.loadTemplates()
	if err != nil {
		return err
	}

	m.rl = NewRateLimiter(m.r, "ratelimit", "ratelimit:total")
	return m.rl.Init(*m.conf.RateLimit)
}

func (m *Manager) Start() error {
	go m.findNotRunningLoop()
	return m.pollLoop()
}

func (m *Manager) ID() string {
	return m.managerID
}
