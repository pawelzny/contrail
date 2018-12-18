#!/usr/bin/env python
import yaml

docker_compose_path = "/etc/contrail/control/docker-compose.yaml"

with open(docker_compose_path) as f:
    docker_compose = yaml.load(f)

docker_compose["services"]["control"]["image"] = "katrybacka/contrail-controller-control-control:etcd_sync"
docker_compose["services"]["control"]["volumes"].append("/etc/kubernetes/pki/etcd:/etc/kubernetes/pki/etcd:ro")

with open(docker_compose_path, "w") as f:
    yaml.dump(docker_compose, f)
