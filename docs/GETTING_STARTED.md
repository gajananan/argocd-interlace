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

Change the following configurations in `./scripts/setup.sh`

1. You will need access to credentials for your registry (they are in a file called image-registry-credentials.json in this example)
OCI_REPOSITORY="gcr.io/your-image-registry"
OCI_IMAGE_PREFIX="someprefix"
OCI_IMAGE_TAG="sometag"
OCI_REGISRY_EMAIL="your-email@gmail.com"
OCI_CREDENTIALS_PATH="/home/image-registry-crendtials.json"

2. You will need access to credentials for your argocd deployment. 
ARGOCD_TOKEN="XXXXXXXX"
ARGOCD_API_BASE_URL="https://argo-route-argocd.apps.<cluster-host-name>/api/v1/applications"

```
./scripts/setup.sh
```


### Install Argocd Interlace

Execute the following command to deploy ArgoCD Interlace to the cluster where  ArgoCD is deployed.

```
kustomize build deploy | kubectl apply -f -
```

### Usecase

1. Fork the following helloworld sample applicaiton repository in your GitHub.

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

3. Check Argocd Interlace log

You can check ArgoCD Interlace log with the following command (`argocd-interlace-controller-65fb7fc9c6-4f2p9` is the pod name of ArgoCD Interlace in this exmaple).

```
time="2021-08-23T08:06:50Z" level=info msg="Starting argocd-interlace..."
time="2021-08-23T08:06:50Z" level=info msg="Synchronizing events..."
time="2021-08-23T08:06:50Z" level=info msg="Synchronization complete!"
time="2021-08-23T08:06:50Z" level=info msg="Ready to process events"
time="2021-08-23T08:07:26Z" level=info msg="manifestStorageType oci"
time="2021-08-23T08:07:27Z" level=info msg="manifestGenerated true"
time="2021-08-23T08:07:27Z" level=info msg="Storing manifest in OCI: gcr.io/some-image-registry/sometag-app-helloworld:mnf "
Uploading file from [/tmp/kubectl-sigstore-temp-dir858520133/manifest.yaml] to [gcr.io/some-image-registry/sometag-app-helloworld:mnf] with media type [application/x-gzip]
File [/tmp/kubectl-sigstore-temp-dir858520133/manifest.yaml] is available directly at [gcr.io/v2/some-image-registry/sometag-app-helloworld/blobs/sha256:63db9f4a38d7f9d29e37a5a64e482bff6bee174cb47ccad22b78fc0d0a4a2372]
Enter password for private key:
Pushing signature to: gcr.io/some-image-registry/argocd.apps.ma4kmc2-app-helloworld:sha256-ae975bea23c2fa358b2d1f766524454150e82f25cb3b8a79df03c399647edaec.sig
time="2021-08-23T08:07:36Z" level=info msg="Storing manifest provenance for OCI: gcr.io/some-image-registry/sometag-app-helloworld:mnf "
time="2021-08-23T08:07:37Z" level=info msg="targetDigest ae975bea23c2fa358b2d1f766524454150e82f25cb3b8a79df03c399647edaec"
time="2021-08-23T08:07:38Z" level=info msg="Created entry at index 674513, available at: https://rekor.sigstore.dev/api/v1/log/entries/b9a43918848ca6c96533e7b6aca5c4b401c46c426a193dd07d988a4d6fd4e960\n"
time="2021-08-23T08:07:38Z" level=info msg="Uploaded attestation to tlog,  uuid: b9a43918848ca6c96533e7b6aca5c4b401c46c426a193dd07d988a4d6fd4e960"
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


