# KubeWatch

Watches docker for new images. Kills any kubernetes pods using old
versions of those images. This way, the Replication Controller will
recreate those pods with the new version.

Useful for developing. For example, if you have a local kubernetes
instance, and have `make` set up to build your docker images, the
images will immediately be auto-deployed.

## Usage

Simply run on your minikube vm with:

    $ kubectl apply -f 'https://raw.githubusercontent.com/paulbellamy/kubewatch/master/k8s-deployment.yaml'

Then everytime you do a `docker build`, your images will be
auto-deployed.

## Note

This will *not* currently work for any multi-host kubernetes setup. It is assumed to be running on minikube.
