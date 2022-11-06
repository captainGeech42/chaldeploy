# chaldeploy

Web app to deploy a CTF challenge to k8s for single-team instance management

![Screenshot of web app](./screenshot.png)

## Features

* Authenticate a team via rCTF, restricting each team to only a single deployment at a time
* Deploy a challenge to a Kubernetes cluster and provide the team with a service endpoint to interact with it
  * k8s config based on the deployments performed by [rCDS](https://github.com/redpwn/rcds/tree/master/rcds/backends/k8s)
* Automatic challenge deletion after a timeout period
  * Teams can extend this if desired

**NOTE**: chaldeploy currently only supports deploying to GKE clusters.

## Usage

You need to set the following environment variables:

* `$CHALDEPLOY_NAME`
  * Name of the challenge to deploy
  * ex: `My First Pwn`
* `$CHALDEPLOY_PORT`
  * Port exposed by the challenge
  * ex: `12345`
* `$CHALDEPLOY_IMAGE`
  * Image path for the challenge
  * ex: `myfirstpwn:latest`
* `$CHALDEPLOY_SESSION_KEY`
  * Secret key used to authenticate session data. Must be 32 or 64 chars long
  * ex: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`
* `$CHALDEPLOY_RCTF_SERVER`
  * rCTF server to auth against
  * ex: `https://2021.redpwn.net`
* `$CHALDEPLOY_K8SCONFIG` (optional)
  * Path to the k8s config. If not set, k8s config will be loaded from /var/run/secrets or ~/.kube
  * ex: `/home/user/specialconfig`

## k8s deployment

TODO: set env vars

```bash
# to do the initial deployment
kubectl apply -f deployment.yaml

# to expose service from minikube
minikube service chaldeploy --url

# to view service logs
kubectl logs -l app=chaldeploy -f

# to deploy a new service version
#   - bump version number in image tag in deployment.yaml
#   - and edit tag in below commands accordingly
export CHALDEP_VER=v1
docker build -t chaldeploy:$CHALDEP_VER . && minikube image load chaldeploy:$CHALDEP_VER
kubectl apply -f deployment.yaml
```

## target app

[src](https://gitlab.com/osusec/ctf-authors/damctf2020-chals/-/tree/master/test/test-nc)

```
docker build -t test-nc:v2
minikube image load test-nc:v2
kubectl apply -f target-app.yaml
```