# KubeWatch

Watches docker for new images. Kills any kubernetes pods using old
versions of those images. This way, the Replication Controller will
recreate those pods with the new version.

Useful for developing. For example, if you have a local kubernetes
instance, and have `make` set up to build your docker images, the
images will immediately be auto-deployed.

## Usage

First, you need to get the CLI tool:

    $ go get -u github.com/paulbellamy/kubewatch

Set up kubernetes, and deploy your replication controllers using the
"latest" image version.

Start kubewatch:

    $ kubewatch

This will connect to Kubernetes using kubectl's currently set context,
and to Docker via the configured environment variables.

Then everytime you do a `docker build`, your images will be
auto-deployed.
