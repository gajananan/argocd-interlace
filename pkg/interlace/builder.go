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

package interlace

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/IBM/argocd-interlace/pkg/application"
	"github.com/IBM/argocd-interlace/pkg/config"
	"github.com/IBM/argocd-interlace/pkg/manifest"
	helmprov "github.com/IBM/argocd-interlace/pkg/provenance/helm"
	"github.com/IBM/argocd-interlace/pkg/provenance/kustomize"
	kustprov "github.com/IBM/argocd-interlace/pkg/provenance/kustomize"
	"github.com/IBM/argocd-interlace/pkg/storage"
	"github.com/IBM/argocd-interlace/pkg/storage/annotation"
	"github.com/IBM/argocd-interlace/pkg/utils"
	appv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
)

func CreateEventHandler(app *appv1.Application) error {

	appName := app.ObjectMeta.Name
	appClusterUrl := app.Spec.Destination.Server

	// Do not use app.Status  in create event.
	appSourceRepoUrl := app.Spec.Source.RepoURL
	appSourceRevision := app.Spec.Source.TargetRevision
	appSourceCommitSha := ""
	// Create does not have app.Status.Sync.Revision information, we need to extract commitsha by API
	commitSha := kustomize.GitLatestCommitSha(app.Spec.Source.RepoURL, app.Spec.Source.TargetRevision)
	if commitSha != "" {
		appSourceCommitSha = commitSha
	}

	log.Infof("[INFO][%s]: Interlace detected creation of new Application resource: %s", appName, appName)
	appPath := ""
	isHelm := app.Spec.Source.IsHelm()
	if isHelm {
		appPath = fmt.Sprintf("%s/%s", "/tmp", appName)
	} else {
		appPath = app.Spec.Source.Path
	}

	appSourcePreiviousCommitSha := ""
	var err error
	sourceVerified := false

	appDirPath := filepath.Join(utils.TMP_DIR, appName, appPath)
	chart := app.Spec.Source.Chart
	appData, _ := application.NewApplicationData(appName, appPath, appDirPath, appClusterUrl,
		appSourceRepoUrl, appSourceRevision, appSourceCommitSha, appSourcePreiviousCommitSha,
		chart, isHelm)

	if isHelm {
		log.Infof("[INFO][%s]: Interlace detected creation of new Application resource: %s", appName, appName)
		prov, _ := helmprov.NewProvenance(*appData)
		sourceVerified, err = prov.VerifySourceMaterial()
		if err != nil {
			log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials failed: %s", appName, appName)
			return err
		}
	} else {
		prov, _ := kustprov.NewProvenance(*appData)
		sourceVerified, err = prov.VerifySourceMaterial()

		if err != nil {
			log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials failed: %s", appName, appName)
			return err
		}
	}
	log.Info("[INFO] sourceVerified ", sourceVerified)
	if sourceVerified {
		log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials succeeded: %s", appName, appName)

		err = signManifestAndGenerateProvenance(*appData, true)

		if err != nil {
			return err
		}
	}
	return nil
}

// Handles update events for the Application CRD
// Triggers the following steps:
// Retrive latest manifest via ArgoCD api
// Sign manifest
// Generate provenance record
// Store signed manifest, provenance record in annotation
func UpdateEventHandler(oldApp, newApp *appv1.Application) error {

	generateManifest := false
	created := false

	if oldApp.Status.OperationState != nil &&
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
		appClusterUrl := newApp.Status.Sync.ComparedTo.Destination.Server
		revisionHistories := newApp.Status.History
		appSourcePreiviousCommitSha := ""
		if revisionHistories != nil {
			log.Info("revisionHistories ", revisionHistories)
			log.Info("history ", len(revisionHistories))
			log.Info("previous revision: ", revisionHistories[len(revisionHistories)-1])
			appSourcePreiviousCommit := revisionHistories[len(revisionHistories)-1]
			appSourcePreiviousCommitSha = appSourcePreiviousCommit.Revision
		}
		var err error
		sourceVerified := false

		log.Infof("[INFO][%s]: Interlace detected update of existing Application resource: %s", appName, appName)
		isHelm := newApp.Spec.Source.IsHelm()
		if isHelm {
			appPath = fmt.Sprintf("%s/%s", "/tmp", appName)
		} else {
			appPath = newApp.Spec.Source.Path
		}

		appDirPath := filepath.Join(utils.TMP_DIR, appName, appPath)
		chart := newApp.Spec.Source.Chart
		appData, _ := application.NewApplicationData(appName, appPath, appDirPath, appClusterUrl,
			appSourceRepoUrl, appSourceRevision, appSourceCommitSha, appSourcePreiviousCommitSha,
			chart, isHelm)

		if isHelm {

			log.Infof("[INFO][%s]: Interlace detected creation of new Application resource: %s", appName, appName)
			prov, _ := helmprov.NewProvenance(*appData)
			sourceVerified, err = prov.VerifySourceMaterial()
			if err != nil {
				log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials failed: %s", appName, appName)
				return err
			}
		} else {
			prov, _ := kustprov.NewProvenance(*appData)
			sourceVerified, err = prov.VerifySourceMaterial()

			if err != nil {
				log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials failed: %s", appName, appName)
				return err
			}
		}

		log.Info("[INFO] sourceVerified ", sourceVerified)
		if sourceVerified {
			log.Infof("[INFO][%s]: Interlace's signature verification of Application source materials succeeded: %s", appName, appName)

			err := signManifestAndGenerateProvenance(*appData, created)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func signManifestAndGenerateProvenance(appData application.ApplicationData, created bool) error {

	interlaceConfig, err := config.GetInterlaceConfig()
	if err != nil {
		log.Errorf("Error in loading config: %s", err.Error())
		return nil
	}

	manifestStorageType := interlaceConfig.ManifestStorageType

	allStorageBackEnds, err := storage.InitializeStorageBackends(appData, manifestStorageType)

	if err != nil {
		log.Errorf("Error in initializing storage backends: %s", err.Error())
		return err
	}

	storageBackend := allStorageBackEnds[manifestStorageType]
	log.Info("manifestStorageType ", manifestStorageType)
	log.Info("storageBackend ", storageBackend)
	if storageBackend != nil {

		manifestGenerated := false

		loc, _ := time.LoadLocation("UTC")
		buildStartedOn := time.Now().In(loc)

		log.Info("buildStartedOn:", buildStartedOn, " loc ", loc)

		if created {
			log.Info("created scenario")
			log.Infof("[INFO][%s] Interlace downloads desired manifest from ArgoCD REST API", appData.AppName)
			manifestGenerated, err = manifest.GenerateInitialManifest(appData)
			if err != nil {
				log.Errorf("Error in generating initial manifest: %s", err.Error())
				return err
			}
		} else {
			log.Info("update scenario")
			log.Infof("[INFO][%s] Interlace downloads desired manifest from ArgoCD REST API", appData.AppName)
			yamlBytes, err := storageBackend.GetLatestManifestContent()
			if err != nil {
				log.Errorf("Error in retriving latest manifest content: %s", err.Error())

				if storageBackend.Type() == annotation.StorageBackendAnnotation {
					log.Info("Going to try generating initial manifest again")
					manifestGenerated, err = manifest.GenerateInitialManifest(appData)
					log.Info("manifestGenerated after generating initial manifest again: ", manifestGenerated)
					if err != nil {
						log.Errorf("Error in generating initial manifest: %s", err.Error())
						return err
					}
				} else {
					return err
				}

			}
			log.Infof("[INFO]: Argocd Interlace generates manifest %s", appData.AppName)
			manifestGenerated, err = manifest.GenerateManifest(appData, yamlBytes)
			if err != nil {
				log.Errorf("Error in generating latest manifest: %s", err.Error())
				return err
			}
		}
		log.Info("manifestGenerated ", manifestGenerated)
		if manifestGenerated {

			err = storageBackend.StoreManifestBundle()
			if err != nil {
				log.Errorf("Error in storing latest manifest bundle(signature, prov) %s", err.Error())
				return err
			}

		}

		buildFinishedOn := time.Now().In(loc)

		log.Info("buildFinishedOn:", buildFinishedOn, " loc ", loc)

		if interlaceConfig.AlwaysGenerateProv {
			if !appData.IsHelm {
				err = storageBackend.StoreManifestProvenance(buildStartedOn, buildFinishedOn)
				if err != nil {
					log.Errorf("Error in storing manifest provenance: %s", err.Error())
					return err
				}
			}
		} else {
			if manifestGenerated && !appData.IsHelm {
				err = storageBackend.StoreManifestProvenance(buildStartedOn, buildFinishedOn)
				if err != nil {
					log.Errorf("Error in storing manifest provenance: %s", err.Error())
					return err
				}
			}
		}
	} else {

		return fmt.Errorf("Could not find storage backend")
	}

	return nil
}
