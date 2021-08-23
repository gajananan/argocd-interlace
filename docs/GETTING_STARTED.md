# Getting started with ArgoCD Interlace

## Prerequisites
- ArgoCD already deployed in a cluster

## Install

### Retrive the source from `ArgoCD Interlace` Git repository.

git clone this repository and moved to `argocd-interlace` directory

```
$ git clone https://github.com/IBM/argocd-interlace.git
$ cd argocd-interlace
$ pwd /home/repo/argocd-interlace
```

### Prepare namespace for installing ArgoCD Interlace

```
kubectl create ns argocd-interlace

```

### Setup secrets

1. You will need access to credentials for your registry (they are in a file called image-registry-credentials.json in this example)

Change env setting `OCI_IMAGE_REGISTRY` in deploy/patch.yaml to your OCI image registry ("gcr.io/your-image-registry").

To access your image registry from ArgoCD Interlacer,  setup a secret in namespace `argocd-interlace` with credentials as below. For example, if your OCI image registry is hosted in Google cloud, refer to (here)[https://cloud.google.com/docs/authentication/getting-started] for setting up acccess credentials.

OCI_IMAGE_REGITSRY_EMAIL="your-email@gmail.com"
OCI_CREDENTIALS_PATH="/home/image-registry-crendtials.json"

```
kubectl create secret docker-registry argocd-interlace-gcr-secret\
 --docker-server "https://gcr.io" --docker-username _json_key\
 --docker-email "$OCI_IMAGE_REGITSRY_EMAIL"\
 --docker-password="$(cat ${OCI_CREDENTIALS_PATH} | jq -c .)"\
 -n argocd-interlace
```

2. You will need access to credentials for your ArgoCD deployment. 

Create a secret that contains `ARGOCD_TOKEN` and `ARGOCD_API_BASE_URL` to create access to your ArgoCD REST API

u

```
export ARGOCD_API_BASE_URL="https://argo-route-argocd.apps.<cluster-host-name>"
export PASSWORD=<>
export ARGOCD_TOKEN=$(curl -k $ARGOCD_SERVER/api/v1/session -d "{\"username\":\"admin\",\"password\": \"$PASSWORD\"}" | jq . -c | jq ."token" | tr -d '"')
```


```
kubectl create secret generic argocd-token-secret\
 --from-literal=ARGOCD_TOKEN=${ARGOCD_TOKEN}\
 --from-literal=ARGOCD_API_BASE_URL=${ARGOCD_API_BASE_URL}\
 -n argocd-interlace
```

3. Creae `cosign` key pairs for creating signatures for generated manifests

```
cosign generate-key-pair
```

### Install Argocd Interlace

Execute the following command to deploy ArgoCD Interlace to the cluster where  ArgoCD is deployed.

```
kustomize build deploy | kubectl apply -f -
namespace/argocd-interlace configured
serviceaccount/argocd-interlace-controller created
clusterrole.rbac.authorization.k8s.io/argocd-interlace-controller-tenant-access created
rolebinding.rbac.authorization.k8s.io/argocd-interlace-controller-tenant-access created
deployment.apps/argocd-interlace-controller created

```
You can check after the successful deployment of ArgoCD Interlace as follows.

```
kubectl get all -n argocd-interlace
NAME                                              READY   STATUS    RESTARTS   AGE
pod/argocd-interlace-controller-f57fd69fb-72l4h   1/1     Running   0          19m

NAME                                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/argocd-interlace-controller   1/1     1            1           19m

NAME                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/argocd-interlace-controller-f57fd69fb   1         1         1       19m
```



### Usecase

Check how ArogCD Interlacer work by using a sample application.  

1. Use the following helloworld sample applicaiton.

https://github.com/kubernetes-sigs/kustomize/tree/master/examples/helloWorld

2.  Confgure helloworld sample applicaiton in your ArgoCD deployment

E.g.: application-helloworld.yaml

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: app-helloworld
  namespace: argocd
spec:
  destination:
    namespace: helloworld-ns
    server: <your-cluster>
  project: default
  source:
    path: examples/helloWorld/
    repoURL: https://github.com/<your-org>/kustomize
    targetRevision: master
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

Create application with the folllowing command
```
kubectl create -n argocd -f application-helloworld.yaml
```


3.  You can find manifest image with signature in the OCI registry

```
gcr.io/some-image-registry/<image-prefix>-app-helloworld:<sometag>
```


4.  You can find provenance record as follows

From Argocd Interlace log, you can find the UUID of sigstre log, which include the provenance record. Using the UUID,  you can check the provenance record generated.

```
rekor-cli get --uuid=b67ae0ad28ebbf57d168c9623bab5d5295d945815c6b4237b0bee3f0501cf8dc  --format json | jq -r .Attestation | base64 -D | jq .

```


