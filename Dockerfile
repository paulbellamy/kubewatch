FROM scratch
ADD kubewatch /usr/bin/kubewatch
ENTRYPOINT [ "kubewatch" ]
