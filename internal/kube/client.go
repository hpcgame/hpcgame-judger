package kube

import (
	"github.com/lcpu-club/hpcgame-judger/internal/utils"
	"k8s.io/client-go/discovery"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

const DefaultQPS = 180
const DefaultBurst = 250

type Client struct {
	host string
	sm   *utils.SecretManager

	cs *kubernetes.Clientset
	dc *dynamic.DynamicClient

	mapper *restmapper.DeferredDiscoveryRESTMapper
	cDis   discovery.CachedDiscoveryInterface
}

func NewClient(host string, sm *utils.SecretManager) (*Client, error) {
	c := &Client{
		host: host,
		sm:   sm,
	}
	err := c.init()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) init() error {
	rConf := &rest.Config{
		Host:            c.host,
		BearerTokenFile: c.sm.GetTokenFile(),
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: c.sm.GetCAFile(),
		},
		QPS:   DefaultQPS,
		Burst: DefaultBurst,
	}
	cs, err := kubernetes.NewForConfig(rConf)
	if err != nil {
		return err
	}
	dc, err := dynamic.NewForConfig(rConf)
	if err != nil {
		return err
	}

	cachedClient := cached.NewMemCacheClient(cs.Discovery())
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)

	c.cs = cs
	c.dc = dc
	c.cDis = cachedClient
	c.mapper = mapper
	return nil
}

func (c *Client) Client() *kubernetes.Clientset {
	return c.cs
}

func (c *Client) Dynamic() *dynamic.DynamicClient {
	return c.dc
}

func (c *Client) Host() string {
	return c.host
}

func (c *Client) SecretManager() *utils.SecretManager {
	return c.sm
}
