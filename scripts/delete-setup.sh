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



# Delete authentication for ArgoCD interlace
# Delete a Docker config type Kubernetes secret
kubectl delete secret oci-registry-setup-secret\
 -n "$NAMESPACE"
 
# Delete authentication for ArgoCD interlace
# Delete a Docker config type Kubernetes secret
kubectl delete secret argocd-interlace-gcr-secret\
 -n "$NAMESPACE"

# Delete authentication for ArgoCD interlace
# Delete token, base URL for querying argocd via REST API as a Kubernetes secret
kubectl delete secret  argocd-token-secret\
 -n "$NAMESPACE"

# Delete authentication for ArgoCD interlace
# Delete token, base URL for querying argocd via REST API as a Kubernetes secret
kubectl delete secret  signing-secrets\
 -n "$NAMESPACE"




