//
// Copyright 2021 IBM Corporation
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

package config

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

type InterlaceConfig struct {
	LogLevel             string
	ManifestStorageType  string
	ArgocdApiBaseUrl     string
	ArgocdApiToken       string
	OciImageRegistry     string
	OciImagePrefix       string
	OciImageTag          string
	RekorServer          string
	RekorTmpDir          string
	ManifestGitUrl       string
	ManifestGitUserId    string
	ManifestGitUserEmail string
	ManifestGitToken     string
}

var instance *InterlaceConfig

func GetInterlaceConfig() (*InterlaceConfig, error) {
	var err error
	if instance == nil {
		instance, err = newConfig()
		if err != nil {
			log.Errorf("Error in loading config: %s", err.Error())
			return nil, err
		}
	}
	return instance, nil
}

func newConfig() (*InterlaceConfig, error) {
	logLevel := os.Getenv("ARGOCD_INTERLACE_LOG_LEVEL")

	manifestStorageType := os.Getenv("MANIFEST_STORAGE")

	if manifestStorageType == "" {
		return nil, fmt.Errorf("MANIFEST_STORAGE is empty, please specify in configuration !")
	}

	argocdApiBaseUrl := os.Getenv("ARGOCD_API_BASE_URL")
	if argocdApiBaseUrl == "" {
		return nil, fmt.Errorf("ARGOCD_API_BASE_URL is empty, please specify in configuration !")
	}

	argocdApiToken := os.Getenv("ARGOCD_TOKEN")
	if argocdApiToken == "" {
		return nil, fmt.Errorf("ARGOCD_TOKEN is empty, please specify in configuration !")
	}

	config := &InterlaceConfig{
		LogLevel:            logLevel,
		ManifestStorageType: manifestStorageType,
		ArgocdApiBaseUrl:    argocdApiBaseUrl,
		ArgocdApiToken:      argocdApiToken,
	}

	if manifestStorageType == "oci" {

		ociImageRegistry := os.Getenv("IMAGE_REGISTRY")

		if ociImageRegistry == "" {
			return nil, fmt.Errorf("IMAGE_REGISTRY is empty, please specify in configuration !")
		}

		config.OciImageRegistry = ociImageRegistry

		ociImagePrefix := os.Getenv("IMAGE_PREFIX")
		if ociImagePrefix == "" {
			return nil, fmt.Errorf("IMAGE_PREFIX is empty, please specify in configuration !")
		}
		config.OciImagePrefix = ociImagePrefix

		ociImageTag := os.Getenv("IMAGE_TAG")
		if ociImageTag == "" {
			return nil, fmt.Errorf("IMAGE_TAG is empty, please specify in configuration !")
		}
		config.OciImageTag = ociImageTag

		rekorServer := os.Getenv("REKOR_SERVER")
		if rekorServer == "" {
			return nil, fmt.Errorf("REKOR_SERVER is empty, please specify in configuration !")
		}
		config.RekorServer = rekorServer

		config.RekorTmpDir = os.Getenv("REKORTMPDIR")

		return config, nil

	} else if manifestStorageType == "git" {

		manifestGitUrl := os.Getenv("MANIFEST_GITREPO_URL")

		if manifestGitUrl == "" {
			return nil, fmt.Errorf("MANIFEST_GITREPO_URL is empty, please specify in configuration !")
		}
		config.ManifestGitUrl = manifestGitUrl

		manifestGitUserId := os.Getenv("MANIFEST_GITREPO_USER")

		if manifestGitUserId == "" {
			return nil, fmt.Errorf("MANIFEST_GITREPO_USER is empty, please specify in configuration !")
		}
		config.ManifestGitUserId = manifestGitUserId

		manifestGitUserEmail := os.Getenv("MANIFEST_GITREPO_USEREMAIL")

		if manifestGitUserEmail == "" {
			return nil, fmt.Errorf("MANIFEST_GITREPO_USEREMAIL is empty, please specify in configuration !")
		}
		config.ManifestGitUserEmail = manifestGitUserEmail

		manifestGitToken := os.Getenv("MANIFEST_GITREPO_TOKEN")

		if manifestGitToken == "" {
			return nil, fmt.Errorf("MANIFEST_GITREPO_TOKEN is empty, please specify in configuration !")
		}
		config.ManifestGitToken = manifestGitToken

		return config, nil

	}

	return nil, fmt.Errorf("Unsupported storage type %s", manifestStorageType)

	/*
		logLevelStr := os.Getenv("ARGOCD_INTERLACE_LOG_LEVEL")
		manifestStorageType := os.Getenv("MANIFEST_STORAGE")
		manifestRepUrl := os.Getenv("MANIFEST_GITREPO_URL")
		baseUrl := os.Getenv("ARGOCD_API_BASE_URL")
		token := os.Getenv("ARGOCD_TOKEN")

		imageRegistry := os.Getenv("IMAGE_REGISTRY")
		imagePrefix := os.Getenv("IMAGE_PREFIX")
		imageTag := os.Getenv("IMAGE_TAG")

		manifestGitUrl := os.Getenv("MANIFEST_GITREPO_URL")
		manifestGitUserId := os.Getenv("MANIFEST_GITREPO_USER")
		manifestGitUserEmail := os.Getenv("MANIFEST_GITREPO_USEREMAIL")
		manifestGitToken := os.Getenv("MANIFEST_GITREPO_TOKEN")

		rekorServer := os.Getenv("REKOR_SERVER")
	*/
}
