# Getting started with ArgoCD Interlace

## Prerequisites
- ArgoCD already deployed in a cluster


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

### Generate key pairs to be used for signing
e.g. cosign

```
cosign generate-key-pair
```

There would be two keys generated: cosign.key, cosign.pub


### Define secret that holds signing keys

Fill in the secret with values from in cosign.key, cosign.pub.

e.g: signing-secrets.yaml
```yaml

apiVersion: v1
data:
  cosign.key: 
  cosign.pub: 
kind: Secret
metadata:
  name: signing-secrets
  namespace: argocd-interlace
type: Opaque
```

```
kubectl create -f signing-secrets.yaml -n argocd-interlace

```

### Define secret that holds information about git repo that holds signed manifests

e.g. manifest-gitrepo-secret.yaml 

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: manifest-gitrepo-secret
  namespace: argocd-interlace
type: Opaque
stringData:
  MANIFEST_GITREPO_USER: 
  MANIFEST_GITREPO_USEREMAIL: ""
  MANIFEST_GITREPO_URL: 
  MANIFEST_GITREPO_TOKEN: ""
  MANIFEST_GITREPO_TARGET_REVISION: "main"
  MANIFEST_GITREPO_TARGET_NS : "default"
  MANIFEST_GITREPO_ARGO_PROJECT : "default"
  MANIFEST_GITREPO_SUFFIX : "-manifest-sig"
 ``` 

```
kubectl create -f manifest-gitrepo-secret.yaml -n argocd-interlace

```

### Define secret that holds ArgoCD REST API token

e.g. argocd-token-secret.yaml
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: argocd-token-secret
  namespace: argocd-interlace
type: Opaque
stringData:
  ARGOCD_TOKEN: 
  ARGOCD_API_BASE_URL: "https://<cluster-hostname-of-argocd>/api/v1/applications"

```
```
kubectl create -f argocd-token-secret.yaml -n argocd-interlace

```

## Install ArgoCD Interlace
```
kustomize build deploy | kubectl apply -f -
``` 
