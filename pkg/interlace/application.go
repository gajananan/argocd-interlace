//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package interlace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ibm/argocd-interlace/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func createApplication(appName, appPath, server string) (string, error) {

	repoUrl := os.Getenv("MANIFEST_GITREPO_URL")
	targetRevision := os.Getenv("MANIFEST_GITREPO_TARGET_REVISION")
	argocdProj := os.Getenv("MANIFEST_GITREPO_ARGO_PROJECT") //"default"
	destNamespace := os.Getenv("MANIFEST_GITREPO_TARGET_NS") //"default"
	suffix := os.Getenv("MANIFEST_GITREPO_SUFFIX")
	manifestSigAppName := appName + suffix
	argocdNs := "argocd"

	path := filepath.Join(appName, appPath)

	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      manifestSigAppName,
			"namespace": argocdNs,
		},
		"spec": map[string]interface{}{
			"destination": map[string]interface{}{
				"namespace": destNamespace,
				"server":    server,
			},
			"source": map[string]interface{}{
				"path":           path,
				"repoURL":        repoUrl,
				"targetRevision": targetRevision,
			},
			"syncPolicy": map[string]interface{}{
				"automated": map[string]interface{}{
					"prune":    true,
					"selfHeal": true,
				},
			},
			"project": argocdProj,
		},
	}

	baseUrl := os.Getenv("ARGOCD_API_BASE_URL")

	if baseUrl == "" {
		return "", fmt.Errorf("ARGOCD_API_BASE_URL is empty, please specify it in configuration!")
	}

	desiredUrl := fmt.Sprintf("%s?upsert=true&validate=true", baseUrl)

	response, err := utils.QueryAPI(desiredUrl, "POST", data)

	if err != nil {
		log.Errorf("Error in querying ArgoCD api: %s", err.Error())
		return "", err
	}

	return response, nil
}

func updateApplication(appName, appPath, server string) (string, error) {

	repoUrl := os.Getenv("MANIFEST_GITREPO_URL")
	targetRevision := os.Getenv("MANIFEST_GITREPO_TARGET_REVISION")
	argocdProj := os.Getenv("MANIFEST_GITREPO_ARGO_PROJECT") //"default"
	destNamespace := os.Getenv("MANIFEST_GITREPO_TARGET_NS") //"default"
	suffix := os.Getenv("MANIFEST_GITREPO_SUFFIX")
	manifestSigAppName := appName + suffix
	argocdNs := "argocd"

	path := filepath.Join(appName, appPath)

	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      manifestSigAppName,
			"namespace": argocdNs,
		},
		"spec": map[string]interface{}{
			"destination": map[string]interface{}{
				"namespace": destNamespace,
				"server":    server,
			},
			"source": map[string]interface{}{
				"path":           path,
				"repoURL":        repoUrl,
				"targetRevision": targetRevision,
			},
			"syncPolicy": map[string]interface{}{
				"automated": map[string]interface{}{
					"prune":    true,
					"selfHeal": true,
				},
			},
			"project": argocdProj,
		},
	}
	baseUrl := os.Getenv("ARGOCD_API_BASE_URL")

	if baseUrl == "" {
		return "", fmt.Errorf("ARGOCD_API_BASE_URL is empty, please specify it in configuration!")
	}

	desiredUrl := fmt.Sprintf("%s/%s", baseUrl, manifestSigAppName)

	response, err := utils.QueryAPI(desiredUrl, "POST", data)
	if err != nil {
		return "", err
	}

	return response, nil
}

func listApplication(appName string) (string, error) {
	suffix := os.Getenv("MANIFEST_GITREPO_SUFFIX")
	manifestSigAppName := appName + suffix
	baseUrl := os.Getenv("ARGOCD_API_BASE_URL")

	if baseUrl == "" {
		return "", fmt.Errorf("ARGOCD_API_BASE_URL is empty, please specify it in configuration!")
	}

	desiredUrl := fmt.Sprintf("%s/%s", baseUrl, manifestSigAppName)

	response, err := utils.QueryAPI(desiredUrl, "GET", nil)

	if err != nil {
		return "", err
	}
	return response, nil
}
