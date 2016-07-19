package docker

import (
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type Client interface {
	Close() error
	Images() <-chan Image
	Errors() <-chan error
}

type Image struct {
	Name    string
	Version string
}

type client struct {
	*docker.Client
	quit   chan struct{}
	images chan Image
	errors chan error
}

func NewClient(addr string) (Client, error) {
	var (
		c   *docker.Client
		err error
	)
	if addr == "" {
		c, err = docker.NewClientFromEnv()
	} else {
		c, err = docker.NewClient(addr)
	}
	if err != nil {
		return nil, err
	}
	client := &client{
		Client: c,
		quit:   make(chan struct{}),
		images: make(chan Image),
		errors: make(chan error),
	}
	go client.loop()

	return client, nil
}

func (c *client) loop() {
	events := make(chan *docker.APIEvents)
	if err := c.AddEventListener(events); err != nil {
		c.errors <- err
		close(c.errors)
		return
	}
	for {
		select {
		case <-c.quit:
			if err := c.RemoveEventListener(events); err != nil {
				c.errors <- err
			}
			close(c.errors)
			return
		case event := <-events:
			if event.Action == "tag" && event.Type == "image" {
				name := event.Actor.Attributes["name"]
				if strings.HasSuffix(name, ":latest") {
					version := strings.TrimPrefix(event.Actor.ID, "sha256:")
					if len(version) > 8 {
						version = version[:8]
					}
					c.images <- Image{
						Name:    strings.TrimSuffix(name, ":latest"),
						Version: version,
					}
				}
			}
		}
	}
}

func (c *client) Images() <-chan Image {
	return c.images
}

func (c *client) Errors() <-chan error {
	return c.errors
}

func (c *client) Close() error {
	close(c.quit)
	return nil
}
