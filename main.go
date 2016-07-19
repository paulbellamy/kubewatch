package main

import (
	"flag"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/paulbellamy/kubewatch/docker"
	"github.com/paulbellamy/kubewatch/kubernetes"
)

func main() {
	var (
		dockerHost   string
		kubeconfig   string
		resyncPeriod time.Duration
	)
	flag.StringVar(&dockerHost, "docker", "", "docker host, default is from the environment.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig file")
	flag.DurationVar(&resyncPeriod, "resync-period", 10*time.Second, "how often to do a full resync of the kubernetes data")
	flag.Parse()

	// Connect to kubernetes
	kube, err := kubernetes.NewClient(kubeconfig, resyncPeriod)
	if err != nil {
		log.Fatal(err)
	}
	defer kube.Close()
	log.Infof("Connected to kubernetes")

	// Connect to docker
	d, err := docker.NewClient(dockerHost)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()
	log.Infof("Connected to docker")

	errors := d.Errors()
	images := d.Images()
	for {
		select {
		case err, ok := <-errors:
			if !ok {
				return
			}
			if err != nil {
				log.Error(err)
			}
		case image, ok := <-images:
			if !ok {
				return
			}

			log.Infof("New Image: %s:%s", image.Name, image.Version)

			if err := kube.UpdateImage(image.Name, image.Version); err != nil {
				log.Error(err)
			}
		}
	}
}
