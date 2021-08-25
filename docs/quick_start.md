# Getting Started

ArgoCD Interlace runs in parallel to an existing ArgoCD deployment in a plugable manner in a cluster.  

Interlace monitors the trigger from state changes of `Application` resources managed by ArgoCD. 

For an application, when detecting new manifest build by ArgoCD, Interlace retrives the latest manifest via REST API call to ArgoCD server, signs the manifest and store it as OCI image in a registry, record the detail of manifest build such as the source files for the build, the command to produce the manifest for reproducibility. Interlace stores those details as provenance records in [in-toto](https://in-toto.io) format and upload it to [Sigstore](https://sigstore.dev/)log for verification.

### Installation
Prerequisite: Install [ArgoCD](https://argo-cd.readthedocs.io/en/stable/getting_started/) on your cluster before you install ArgoCD Interlace.


To install the latest version of ArgoCD Interlace to your Kubernetes cluster, run:
```
kubectl apply --filename https://raw.githubusercontent.com/IBM/argocd-interlace/main/releases/release.yaml
```

To verify that installation was successful, wait until all Pods have Status `Running`:
```shell
$ kubectl get all -n argocd-interlace
NAME                                              READY   STATUS    RESTARTS   AGE
pod/argocd-interlace-controller-f57fd69fb-72l4h   1/1     Running   0          19m
```

### Setup

To complete setting up ArgoCD Interlace, please follow the steps in [doc](docs/setup.md).
* Add image registry authentication
* Add ArgoCD REST API authentication
* Add cosign based signing keys

to the ArgoCD Interlace controller.


## Tutorial
To get started with ArgoCD Interlace, try out our getting started [tutorial](docs/tutorial.md)!