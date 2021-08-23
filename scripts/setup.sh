#!/bin/bash
#
# Copyright 2020 IBM Corporation.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

OCI_REPOSITORY="gcr.io/kg-image-registry"
OCI_IMAGE_PREFIX="argocd.apps.ma4kmc2"
OCI_IMAGE_TAG="mnf"
OCI_REGISRY_EMAIL="kgajanananan2021@gmail.com"
OCI_CREDENTIALS_PATH="kg-image-registry-9ca89e70732f.json"
ARGOCD_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2Mjk3NzMyOTYsImp0aSI6IjlhZjU4NjMxLTA5ZmUtNDM4MS04ODVlLTc1Mjc2ZGUyNDNlYiIsImlhdCI6MTYyOTY4Njg5NiwiaXNzIjoiYXJnb2NkIiwibmJmIjoxNjI5Njg2ODk2LCJzdWIiOiJhZG1pbjpsb2dpbiJ9.cGCtGwEYj6fwTxYm3MvoTCXD5xiQNCh0Un_oxhLrQFQ"
ARGOCD_API_BASE_URL="https://argo-route-argocd.apps.ma4kmc2.openshiftv4test.com/api/v1/applications"
NAMESPACE="argocd-interlace"
COSIGN_PWD=""

kubectl create secret generic oci-registry-setup-secret\
 --from-literal=IMAGE_REGISTRY=${OCI_REPOSITORY}\
 --from-literal=IMAGE_PREFIX=${OCI_IMAGE_PREFIX}\
 --from-literal=IMAGE_TAG=${OCI_IMAGE_TAG}\
 -n "$NAMESPACE"

# Add authentication for ArgoCD interlace
# Create a Docker config type Kubernetes secret
kubectl create secret docker-registry argocd-interlace-gcr-secret\
 --docker-server "https://gcr.io" --docker-username _json_key\
 --docker-email "$OCI_REGISRY_EMAIL"\
 --docker-password="$(cat ${OCI_CREDENTIALS_PATH} | jq -c .)"\
 -n "$NAMESPACE"

# Add authentication for ArgoCD interlace
# Add token, base URL for querying argocd via REST API as a Kubernetes secret
kubectl create secret generic argocd-token-secret\
 --from-literal=ARGOCD_TOKEN=${ARGOCD_TOKEN}\
 --from-literal=ARGOCD_API_BASE_URL=${ARGOCD_API_BASE_URL}\
 -n "$NAMESPACE"

# Add authentication for ArgoCD interlace
# Add cosign signing key as a Kubernetes secret
echo -n ${COSIGN_PWD} | cosign generate-key-pair "$NAMESPACE"/signing-secrets




