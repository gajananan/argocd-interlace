# Getting started with ArgoCD Interlace

ArgoCD Interlace runs in parallel to an existing ArgoCD deployment in a plugable manner in a cluster.  

Interlace monitors the trigger from state changes of `Application` resources managed by ArgoCD. 

For an application, when detecting new manifest build by ArgoCD, Interlace retrive the latest manifest via REST API call to ArgoCD, signs the manifest and store it as OCI image, record the detail of manifest build such as the source files for the build, the command to produce the manifest for reproducibility. Interlace stores those details as provenance records in [in-toto](https://in-toto.io) format and upload it to [Sigstore](https://sigstore.dev/)log for verification.


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

1. You will need access to credentials for your OCI image registry (they are in a file called image-registry-credentials.json in this example)

For example, if your OCI image registry is hosted in Google cloud, refer to [here](https://cloud.google.com/docs/authentication/getting-started) for setting up acccess credentials.


To access your image registry from ArgoCD Interlacer
- Change env setting `OCI_IMAGE_REGISTRY` in deploy/patch.yaml to your OCI image registry (e.g. "gcr.io/your-image-registry").
- Setup a secret `argocd-interlace-gcr-secret` in namespace `argocd-interlace` with credentials as below. 


Create secret with the following command:
```
OCI_IMAGE_REGITSRY_EMAIL="your-email@gmail.com"
OCI_CREDENTIALS_PATH="/home/image-registry-crendtials.json"

kubectl create secret docker-registry argocd-interlace-gcr-secret\
 --docker-server "https://gcr.io" --docker-username _json_key\
 --docker-email "$OCI_IMAGE_REGITSRY_EMAIL"\
 --docker-password="$(cat ${OCI_CREDENTIALS_PATH} | jq -c .)"\
 -n argocd-interlace
```

2. You will need access to credentials for your ArgoCD deployment. 

Create a secret that contains `ARGOCD_TOKEN` and `ARGOCD_API_BASE_URL` to create access to your ArgoCD REST API.

See [here](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/#local-usersaccounts-v15) for setting up a user account with readonly access in ArgoCD

A sample set of steps to create user account with readonly access and to retrive `ARGOCD_TOKEN` in ArgoCD is described [here](./SETUP_ARGOCD_USER_ACCOUNT.md)

Retrive a token for your user account in ArgoCD

```
export ARGOCD_API_BASE_URL="https://argo-route-argocd.apps.<cluster-host-name>"
export PASSWORD=<>
export ARGOCD_TOKEN=$(curl -k $ARGOCD_SERVER/api/v1/session -d "{\"username\":\"admin\",\"password\": \"$PASSWORD\"}" | jq . -c | jq ."token" | tr -d '"')
```

Create a secret with the retrived token and base url:
```
kubectl create secret generic argocd-token-secret\
 --from-literal=ARGOCD_TOKEN=${ARGOCD_TOKEN}\
 --from-literal=ARGOCD_API_BASE_URL=${ARGOCD_API_BASE_URL}\
 -n argocd-interlace
```

3. Create `cosign` key pairs for creating signatures for generated manifests by ArgoCD Interlace

You will need to setup a key pair for signing manifest 
https://github.com/sigstore/cosign


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

When creating a new application,  Argocd Interlacer monitors the trigger from state changes of Application resources on the ArgoCD cluster.

When detecting new manifest build, Interlace sign the manifest, record the detail of manifest build such as the source files for the build, the command to produce the manifest for reproducibility. Interlace stores those details as provenance records in in-toto format. 


3.  You can find manifest image with signature in the OCI registry

```
gcr.io/some-image-registry/<image-prefix>-app-helloworld:<sometag>
```


4.  You can find provenance record generated by ArgoCD Interlacer as follows

From Argocd Interlace log, you can find the UUID of sigstre log, which include the provenance record. Using the UUID,  you can check the provenance record generated.

```
rekor-cli get --uuid=b67ae0ad28ebbf57d168c9623bab5d5295d945815c6b4237b0bee3f0501cf8dc  --format json | jq -r .Attestation | base64 -D | jq .

```


