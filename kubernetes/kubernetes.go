package kubernetes

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	k8sclient "k8s.io/client-go/1.5/kubernetes"
	v1core "k8s.io/client-go/1.5/kubernetes/typed/core/v1"
	api "k8s.io/client-go/1.5/pkg/api"
	v1 "k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/tools/cache"
	"k8s.io/client-go/1.5/tools/clientcmd"
)

// Client keeps track of running kubernetes pods and services
type Client interface {
	Close() error
	UpdateImage(name, version string) error
}

type client struct {
	quit         chan struct{}
	resyncPeriod time.Duration
	client       v1core.CoreInterface
	store        cache.Store
}

// NewClient returns a usable Client. Don't forget to Close it.
func NewClient(filename string, resyncPeriod time.Duration) (Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	// TODO: handle the filename for kubeconfig here, as well.
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}
	config.ContentConfig.GroupVersion = &api.Unversioned

	c, err := k8sclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "starting corev1 client")
	}
	core := c.Core()

	// This will hold the downstream state, as we know it.
	store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	// Create the controller and run it until we close stop.
	quit := make(chan struct{})
	cache.NewReflector(
		cache.NewListWatchFromClient(
			core.GetRESTClient(),
			"pods",
			"",
			fields.Everything(),
		),
		&v1.Pod{},
		store,
		resyncPeriod,
	).RunUntil(quit)

	return &client{
		quit:         quit,
		resyncPeriod: resyncPeriod,
		client:       core,
		store:        store,
	}, nil
}

// UpdateImage kills all pods using a different version of this image, so that
// they will be re-created with the new one.
func (c *client) UpdateImage(name, version string) error {
	for _, podInterface := range c.store.List() {
		pod := podInterface.(*v1.Pod)
		containers := pod.Spec.Containers
		found := false
		for _, c := range containers {
			i, v := parseImage(c.Image)
			if i == name && v == "latest" {
				found = true
				break
			}
		}
		if found {
			if err := c.client.Pods(pod.ObjectMeta.Namespace).Delete(pod.ObjectMeta.Name, nil); err != nil {
				return err
			}
			logDeployment(name, version, pod.TypeMeta.Kind, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		}
	}
	return nil
}

func parseImage(image string) (name, version string) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "latest"
}

func logDeployment(image, version, kind, namespace, name string) {
	itemName := fmt.Sprintf("%s/%s", namespace, name)
	if kind != "" {
		itemName = kind + ":" + itemName
	}
	log.Infof("Deployed: %s:%s -> %s", image, version, itemName)
}

func (c *client) Close() error {
	close(c.quit)
	return nil
}
