package kubernetes

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	unversionedAPI "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/wait"
)

// Client keeps track of running kubernetes pods and services
type Client interface {
	Close() error
	UpdateImage(name, version string) error
}

type client struct {
	quit             chan struct{}
	resyncPeriod     time.Duration
	client           *unversioned.Client
	extensionsClient *unversioned.ExtensionsClient
	podStore         *cache.StoreToPodLister
}

// runReflectorUntil is equivalent to cache.Reflector.RunUntil, but it also logs
// errors, which cache.Reflector.RunUntil simply ignores
func runReflectorUntil(r *cache.Reflector, resyncPeriod time.Duration, stopCh <-chan struct{}) {
	loggingListAndWatch := func() {
		if err := r.ListAndWatch(stopCh); err != nil {
			log.Errorf("Kubernetes reflector: %v", err)
		}
	}
	go wait.Until(loggingListAndWatch, resyncPeriod, stopCh)
}

// NewClient returns a usable Client. Don't forget to Close it.
func NewClient(filename string, resyncPeriod time.Duration) (Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	// TODO: handle the filename for kubeconfig here, as well.
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	c, err := unversioned.New(config)
	if err != nil {
		return nil, err
	}

	result := &client{
		quit:         make(chan struct{}),
		resyncPeriod: resyncPeriod,
		client:       c,
	}

	result.podStore = &cache.StoreToPodLister{
		Indexer: result.setupStore(c, "pods", &api.Pod{}).(cache.Indexer),
	}

	return result, nil
}

func (c *client) setupStore(kclient cache.Getter, resource string, itemType interface{}) cache.Store {
	lw := cache.NewListWatchFromClient(kclient, resource, api.NamespaceAll, fields.Everything())
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	runReflectorUntil(cache.NewReflector(lw, itemType, store, c.resyncPeriod), c.resyncPeriod, c.quit)
	return store
}

// UpdateImage kills all pods using a different version of this image, so that
// they will be re-created with the new one.
func (c *client) UpdateImage(name, version string) error {
	list, err := c.podStore.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, pod := range list {
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
			logDeployment(name, version, pod.TypeMeta, pod.ObjectMeta)
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

func logDeployment(image, version string, typeMeta unversionedAPI.TypeMeta, objectMeta api.ObjectMeta) {
	itemName := fmt.Sprintf("%s/%s", objectMeta.Namespace, objectMeta.Name)
	if typeMeta.Kind != "" {
		itemName = typeMeta.Kind + ":" + itemName
	}
	log.Infof("Deployed: %s:%s -> %s", image, version, itemName)
}

func (c *client) Close() error {
	close(c.quit)
	return nil
}
