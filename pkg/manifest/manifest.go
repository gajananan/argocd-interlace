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

package manifest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IBM/argocd-interlace/pkg/utils"
	k8smnfutil "github.com/sigstore/k8s-manifest-sigstore/pkg/util"
	"github.com/sigstore/k8s-manifest-sigstore/pkg/util/mapnode"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GenerateInitialManifest(appName, appPath, appDirPath string) (bool, error) {

	// Retrive the desired state of manifest via argocd API call
	desiredManifest, err := utils.RetriveDesiredManifest(appName)
	if err != nil {
		log.Errorf("Error in retriving desired manifest : %s", err.Error())
		return false, err
	}

	items := gjson.Get(desiredManifest, "items")

	finalManifest := ""

	for i, item := range items.Array() {

		targetState := gjson.Get(item.String(), "targetState").String()

		finalManifest = prepareFinalManifest(targetState, finalManifest, i, len(items.Array())-1)
	}

	if finalManifest != "" {

		err := utils.WriteToFile(string(finalManifest), appDirPath, utils.MANIFEST_FILE_NAME)
		if err != nil {
			log.Errorf("Error in writing manifest to file: %s", err.Error())
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func GenerateManifest(appName, appDirPath string, yamlBytes []byte) (bool, error) {

	diffCount := 0
	finalManifest := ""

	manifestYAMLs := k8smnfutil.SplitConcatYAMLs(yamlBytes)

	// Retrive the desired state of manifest via argocd API call
	desiredManifest, err := utils.RetriveDesiredManifest(appName)
	if err != nil {
		log.Errorf("Error in retriving desired manifest : %s", err.Error())
		return false, err
	}

	items := gjson.Get(desiredManifest, "items")

	// For each resource in desired manifest
	// Check if it has changed from the version that exist in the bundle manifest
	for i, item := range items.Array() {
		targetState := gjson.Get(item.String(), "targetState").String()
		if diffCount == 0 {
			diffExist, err := checkDiff([]byte(targetState), manifestYAMLs)
			if err != nil {
				return false, err
			}
			if diffExist {
				diffCount += 1
			}
		}
		// Add desired state of each resource to finalManifest
		finalManifest = prepareFinalManifest(targetState, finalManifest, i, len(items.Array())-1)

	}

	if finalManifest != "" {
		err := utils.WriteToFile(string(finalManifest), appDirPath, utils.MANIFEST_FILE_NAME)
		if err != nil {
			log.Errorf("Error in writing manifest to file: %s", err.Error())
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func checkDiff(targetObjYAMLBytes []byte, manifestYAMLs [][]byte) (bool, error) {

	objNode, err := mapnode.NewFromBytes(targetObjYAMLBytes) // json

	log.Debug("targetObjYAMLBytes ", string(targetObjYAMLBytes))

	if err != nil {
		log.Errorf("objNode error from NewFromYamlBytes %s", err.Error())
		return false, err

	}

	found := false
	for _, manifest := range manifestYAMLs {

		mnfNode, err := mapnode.NewFromYamlBytes(manifest)
		if err != nil {
			log.Errorf("mnfNode error from NewFromYamlBytes %s", err.Error())
			return false, err

		}

		diffs := objNode.Diff(mnfNode)

		// when diffs == nil,  there is no difference in YAMLs being compared.
		if diffs == nil || diffs.Size() == 0 {
			found = true
			break
		}
	}
	return found, nil

}

func prepareFinalManifest(targetState, finalManifest string, counter int, numberOfitems int) string {

	var obj *unstructured.Unstructured

	err := json.Unmarshal([]byte(targetState), &obj)
	if err != nil {
		log.Infof("Error in unmarshaling err %s", err.Error())
	}

	objBytes, _ := yaml.Marshal(obj)
	endLine := ""
	if !strings.HasSuffix(string(objBytes), "\n") {
		endLine = "\n"
	}

	finalManifest = fmt.Sprintf("%s%s%s", finalManifest, string(objBytes), endLine)
	finalManifest = strings.ReplaceAll(finalManifest, "object:\n", "")

	if counter < numberOfitems {
		finalManifest = fmt.Sprintf("%s---\n", finalManifest)
	}

	return finalManifest
}
