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

package annotation

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/IBM/argocd-interlace/pkg/config"
	"github.com/IBM/argocd-interlace/pkg/provenance"
	"github.com/IBM/argocd-interlace/pkg/sign"
	"github.com/IBM/argocd-interlace/pkg/utils"
	"github.com/ghodss/yaml"
	k8smnfutil "github.com/sigstore/k8s-manifest-sigstore/pkg/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StorageBackend struct {
	appName                     string
	appPath                     string
	appDirPath                  string
	appSourceRepoUrl            string
	appSourceRevision           string
	appSourceCommitSha          string
	appSourcePreiviousCommitSha string
	buildStartedOn              time.Time
	buildFinishedOn             time.Time
}

const (
	StorageBackendAnnotation = "annotation"
)

func NewStorageBackend(appName, appPath, appDirPath,
	appSourceRepoUrl, appSourceRevision, appSourceCommitSha, appSourcePreiviousCommitSha string) (*StorageBackend, error) {
	return &StorageBackend{
		appName:                     appName,
		appPath:                     appPath,
		appDirPath:                  appDirPath,
		appSourceRepoUrl:            appSourceRepoUrl,
		appSourceRevision:           appSourceRevision,
		appSourceCommitSha:          appSourceCommitSha,
		appSourcePreiviousCommitSha: appSourcePreiviousCommitSha,
	}, nil
}

func (s StorageBackend) GetLatestManifestContent() ([]byte, error) {
	return nil, nil
}

func (s StorageBackend) StoreManifestBundle(sourceVerifed bool) error {

	keyPath := utils.PRIVATE_KEY_PATH
	manifestPath := filepath.Join(s.appDirPath, utils.MANIFEST_FILE_NAME)
	signedManifestPath := filepath.Join(s.appDirPath, utils.SIGNED_MANIFEST_FILE_NAME)

	signedBytes, err := sign.SignManifest(keyPath, manifestPath, signedManifestPath)

	if err != nil {
		log.Errorf("Error in signing bundle image: %s", err.Error())
		return err
	}

	log.Info("signedBytes: ", string(signedBytes))

	manifestYAMLs := k8smnfutil.SplitConcatYAMLs(signedBytes)
	log.Info("len(manifestYAMLs): ", len(manifestYAMLs))
	var annotations map[string]string
	for _, item := range manifestYAMLs {

		var obj unstructured.Unstructured
		err := yaml.Unmarshal(item, &obj)
		if err != nil {
			log.Errorf("Error unmarshling: %s", err.Error())
		}

		kind := obj.GetKind()
		resourceName := obj.GetName()
		namespace := obj.GetNamespace()
		resourceAnnotatons := obj.GetAnnotations()
		resourceLabels := obj.GetLabels()
		log.Info("kind :", kind, " resourceName ", resourceName, " namespace", namespace)
		log.Info("resourceAnnotatons ", resourceAnnotatons)
		interlaceConfig, err := config.GetInterlaceConfig()
		isSignatureresource := false
		if rscAnnotation, ok := resourceAnnotatons[interlaceConfig.SignatureResourceAnnotation]; ok {
			isSignatureresource, _ = strconv.ParseBool(rscAnnotation)
		} else if rscLabel, ok := resourceLabels[interlaceConfig.SignatureResourceLabel]; ok {
			isSignatureresource, _ = strconv.ParseBool(rscLabel)
		}

		log.Info("isSignatureresource :", isSignatureresource)

		if isSignatureresource {
			log.Info("Going to patch kind:", kind, " name:", resourceName, " in namespace:", namespace)

			annotations = k8smnfutil.GetAnnotationsInYAML(item)

			message := "null"
			signature := "null"
			if sourceVerifed {
				message = annotations[utils.MSG_ANNOTATION_NAME]
				signature = annotations[utils.SIG_ANNOTATION_NAME]
			}

			log.Info("message: ", message)
			log.Info("signature: ", signature)

			patchData, err := preparePatch(message, signature, kind)
			if err != nil {
				log.Errorf("Error in creating patch for application resource config: %s", err.Error())
				return err
			}

			log.Info("len(patchData)", len(patchData))
			log.Info("patchData)", patchData)

			log.Infof("[INFO][%s] Interlace attaches signature to resource as annotation:", s.appName)

			err = utils.ApplyResourcePatch(kind, resourceName, namespace, s.appName, patchData)

			if err != nil {
				log.Errorf("Error in patching application resource config: %s", err.Error())
				return nil
			}

		}

	}

	if err != nil {
		log.Errorf("Error in getting digest: %s ", err.Error())
		return err
	}
	return nil
}

func preparePatch(message, signature, kind string) ([]string, error) {

	var patchData []string
	if kind == "ConfigMap" {

		patchSig := fmt.Sprintf("{\"%s\": {\"%s\": \"%s\"}}",
			"data", "signature", signature)
		patchData = append(patchData, patchSig)
		patchMsg := fmt.Sprintf("{\"%s\": {\"%s\": \"%s\"}}",
			"data", "message", message)
		patchData = append(patchData, patchMsg)
	} else {
		sigAnnot := utils.SIG_ANNOTATION_NAME

		patchSig := fmt.Sprintf("{\"%s\": { \"%s\" : {\"%s\": \"%s\"}}}",
			"metadata", "annotations", sigAnnot, signature)
		patchData = append(patchData, patchSig)

		msgAnnot := utils.MSG_ANNOTATION_NAME
		patchMsg := fmt.Sprintf("{\"%s\": { \"%s\" : {\"%s\": \"%s\"}}}",
			"metadata", "annotations", msgAnnot, message)

		patchData = append(patchData, patchMsg)
	}

	return patchData, nil
}

func (s StorageBackend) StoreManifestProvenance(buildStartedOn time.Time, buildFinishedOn time.Time) error {
	manifestPath := filepath.Join(s.appDirPath, utils.MANIFEST_FILE_NAME)
	computedFileHash, err := utils.ComputeHash(manifestPath)

	err = provenance.GenerateProvanance(s.appName, s.appPath, s.appSourceRepoUrl,
		s.appSourceRevision, s.appSourceCommitSha, s.appSourcePreiviousCommitSha,
		manifestPath, computedFileHash, buildStartedOn, buildFinishedOn, true)
	if err != nil {
		log.Errorf("Error in storing provenance: %s", err.Error())
		return err
	}

	return nil
}

func (b *StorageBackend) Type() string {
	return StorageBackendAnnotation
}
