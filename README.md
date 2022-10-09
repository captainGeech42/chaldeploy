# chaldeploy

Web app to deploy a CTF challenge to k8s for single-team instance management

## k8s deployment

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
docker build -t chaldeploy:v1 .
minikube image load chaldeploy:v1
kubectl apply -f deployment.yaml
```