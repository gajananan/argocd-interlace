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
	"os"
	"path/filepath"
	"strings"
	"time"

	appv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/gajananan/argocd-interlace/pkg/manifest"
	"github.com/gajananan/argocd-interlace/pkg/storage"
	"github.com/gajananan/argocd-interlace/pkg/storage/git"
	"github.com/gajananan/argocd-interlace/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// Handles update events for the Application CRD
// Triggers the following steps:
// Retrive latest manifest via ArgoCD api
// Sign manifest
// Generate provenance record
// Store signed manifest, provenance record in OCI registry/Git
func UpdateEventHandler(oldApp, newApp *appv1.Application) {

	generateManifest := false
	created := false
	if oldApp.Status.Health.Status == "" &&
		oldApp.Status.OperationState != nil &&
		oldApp.Status.OperationState.Phase == "Running" &&
		oldApp.Status.Sync.Status == "" &&
		newApp.Status.Health.Status == "Missing" &&
		newApp.Status.OperationState != nil &&
		newApp.Status.OperationState.Phase == "Running" &&
		newApp.Status.Sync.Status == "OutOfSync" {
		// This branch handle the case in which app is newly created,
		// the follow updates contains the necessary information (commit hash etc.)
		generateManifest = true
		created = true
	} else if oldApp.Status.OperationState != nil &&
		oldApp.Status.OperationState.Phase == "Running" &&
		oldApp.Status.Sync.Status == "Synced" &&
		newApp.Status.OperationState != nil &&
		newApp.Status.OperationState.Phase == "Running" &&
		newApp.Status.Sync.Status == "OutOfSync" {
		// This branch handle the case in which app is being updated,
		// the updates contains the necessary information (commit hash etc.)
		generateManifest = true
	}

	if generateManifest {

		appName := newApp.ObjectMeta.Name
		appPath := newApp.Status.Sync.ComparedTo.Source.Path
		appSourceRepoUrl := newApp.Status.Sync.ComparedTo.Source.RepoURL
		appSourceRevision := newApp.Status.Sync.ComparedTo.Source.TargetRevision
		appSourceCommitSha := newApp.Status.Sync.Revision
		appServer := newApp.Status.Sync.ComparedTo.Destination.Server

		signManifestAndGenerateProvenance(appName, appPath, appServer,
			appSourceRepoUrl, appSourceRevision, appSourceCommitSha, created,
		)

	}

}

func signManifestAndGenerateProvenance(appName, appPath, appServer,
	appSourceRepoUrl, appSourceRevision, appSourceCommitSha string, created bool) {

	appDirPath := filepath.Join(utils.TMP_DIR, appName, appPath)

	manifestGitUrl := os.Getenv("MANIFEST_GITREPO_URL")

	if manifestGitUrl == "" {
		log.Error("MANIFEST_GITREPO_URL is empty, please specify in configuration !")
		return
	}

	manifestGitUserId := os.Getenv("MANIFEST_GITREPO_USER")

	if manifestGitUserId == "" {
		log.Error("MANIFEST_GITREPO_USER is empty, please specify in configuration !")
		return
	}

	manifestGitUserEmail := os.Getenv("MANIFEST_GITREPO_USEREMAIL")

	if manifestGitUserEmail == "" {
		log.Error("MANIFEST_GITREPO_USEREMAIL is empty, please specify in configuration !")
		return
	}

	manifestGitToken := os.Getenv("MANIFEST_GITREPO_TOKEN")

	if manifestGitToken == "" {
		log.Error("MANIFEST_GITREPO_TOKEN is empty, please specify in configuration !")
		return
	}

	allStorageBackEnds, err := storage.InitializeStorageBackends(appName, appPath, appDirPath,
		appSourceRepoUrl, appSourceRevision, appSourceCommitSha,
		manifestGitUrl, manifestGitUserId, manifestGitUserEmail, manifestGitToken,
	)

	if err != nil {
		log.Errorf("Error in initializing storage backends: %s", err.Error())
		return
	}

	for _, storageBackend := range allStorageBackEnds {

		manifestGenerated := false

		loc, _ := time.LoadLocation("UTC")
		buildStartedOn := time.Now().In(loc)
		storageBackend.SetBuildStartedOn(buildStartedOn)

		if created {
			manifestGenerated, err = manifest.GenerateInitialManifest(appName, appPath, appDirPath)
			if err != nil {
				log.Errorf("Error in generating initial manifest %s", err.Error())
				continue
			}
		} else {
			yamlBytes, err := storageBackend.GetLatestManifestContent()
			if err != nil {
				log.Errorf("Error in retriving latest manifest content %s", err.Error())
				continue
			}
			manifestGenerated, err = manifest.GenerateManifest(appName, appDirPath, yamlBytes)
			if err != nil {
				log.Errorf("Error in generating latest manifest %s", err.Error())
				continue
			}
		}

		if manifestGenerated {

			err = storageBackend.StoreManifestSignature()
			if err != nil {
				log.Errorf("Error in storing latest manifest signature %s", err.Error())
				continue
			}

			if storageBackend.Type() == git.StorageBackendGit {

				response := listApplication(appName)

				if strings.Contains(response, "not found") {
					createApplication(appName, appPath, appServer)
				} else {
					updateApplication(appName, appPath, appServer)
				}

			}
			buildFinishedOn := time.Now().In(loc)
			storageBackend.SetBuildFinishedOn(buildFinishedOn)

			err = storageBackend.StoreManifestProvenance()
			if err != nil {
				log.Errorf("Error in storing latest manifest provenance %s", err.Error())
				continue
			}
		}
	}

	return
}
