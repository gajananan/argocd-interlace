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

package helm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/argocd-interlace/pkg/application"
	"github.com/IBM/argocd-interlace/pkg/config"
	"github.com/IBM/argocd-interlace/pkg/provenance/attestation"
	"github.com/IBM/argocd-interlace/pkg/utils"
	"github.com/in-toto/in-toto-golang/in_toto"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type Provenance struct {
	appData application.ApplicationData
}

const (
	ProvenanceAnnotation = "helm"
)

func NewProvenance(appData application.ApplicationData) (*Provenance, error) {
	return &Provenance{
		appData: appData,
	}, nil
}

func (p Provenance) GenerateProvanance(target, targetDigest string, uploadTLog bool, buildStartedOn time.Time, buildFinishedOn time.Time) error {
	appName := p.appData.AppName
	appPath := p.appData.AppPath
	appSourceRepoUrl := p.appData.AppSourceRepoUrl
	appSourceRevision := p.appData.AppSourceRevision
	appSourceCommitSha := p.appData.AppSourceCommitSha
	appDirPath := p.appData.AppDirPath
	chart := p.appData.Chart
	interlaceConfig, err := config.GetInterlaceConfig()
	argocdNamespace := interlaceConfig.ArgocdNamespace
	entryPoint := "argocd-interlace"
	recipe := in_toto.ProvenanceRecipe{
		EntryPoint: entryPoint,
		Arguments:  []string{"-n " + argocdNamespace},
	}

	if err != nil {
		log.Infof("err in prov: %s ", err.Error())
	}

	subjects := []in_toto.Subject{}

	targetDigest = strings.ReplaceAll(targetDigest, "sha256:", "")
	subjects = append(subjects, in_toto.Subject{Name: target,
		Digest: in_toto.DigestSet{
			"sha256": targetDigest,
		},
	})

	materials := generateMaterial(appName, appPath, appSourceRepoUrl, appSourceRevision,
		appSourceCommitSha, chart, "")

	it := in_toto.Statement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: in_toto.PredicateSLSAProvenanceV01,
			Subject:       subjects,
		},
		Predicate: in_toto.ProvenancePredicate{
			Metadata: &in_toto.ProvenanceMetadata{
				Reproducible:    true,
				BuildStartedOn:  &buildStartedOn,
				BuildFinishedOn: &buildFinishedOn,
			},

			Materials: materials,
			Recipe:    recipe,
		},
	}
	b, err := json.Marshal(it)
	if err != nil {
		log.Errorf("Error in marshaling attestation:  %s", err.Error())
		return err
	}

	err = utils.WriteToFile(string(b), appDirPath, utils.PROVENANCE_FILE_NAME)
	if err != nil {
		log.Errorf("Error in writing provenance to a file:  %s", err.Error())
		return err
	}

	err = attestation.GenerateSignedAttestation(it, appName, appDirPath, uploadTLog)
	if err != nil {
		log.Errorf("Error in generating signed attestation:  %s", err.Error())
		return err
	}

	return nil
}

func generateMaterial(appName, appPath, appSourceRepoUrl, appSourceRevision, appSourceCommitSha, chart string, provTrace string) []in_toto.ProvenanceMaterial {

	materials := []in_toto.ProvenanceMaterial{}

	chartHash, _ := getSha256sum(appPath, chart, appSourceRevision)

	materials = append(materials, in_toto.ProvenanceMaterial{
		URI: appSourceRepoUrl + ".git",
		Digest: in_toto.DigestSet{
			"sha256hash": chartHash,
			"revision":   appSourceRevision,
			"name":       appName,
		},
	})

	appSourceRepoUrlFul := appSourceRepoUrl + ".git"
	materialsStr := gjson.Get(provTrace, "predicate.materials")

	for _, mat := range materialsStr.Array() {

		uri := gjson.Get(mat.String(), "uri").String()
		path := gjson.Get(mat.String(), "digest.path").String()
		revision := gjson.Get(mat.String(), "digest.revision").String()
		commit := gjson.Get(mat.String(), "digest.commit").String()

		if uri != appSourceRepoUrlFul {
			intoMat := in_toto.ProvenanceMaterial{
				URI: uri,
				Digest: in_toto.DigestSet{
					"commit":   commit,
					"revision": revision,
					"path":     path,
				},
			}
			materials = append(materials, intoMat)
		}
	}

	return materials
}

func getSha256sum(appPath, chart, targetRevision string) (string, error) {

	helmChartPath := fmt.Sprintf("%s/%s-%s.tgz", appPath, chart, targetRevision)
	shaCmd := fmt.Sprintf("sha256sum %s | awk '{print $1}'", helmChartPath)
	sha256Hash, err := utils.CmdExec(shaCmd, appPath)
	if err != nil {
		log.Infof("[INFO]: sh256sum CmdExec download : %s ", err.Error())
		return "", err
	}

	return sha256Hash, nil

}

func (p Provenance) VerifySourceMaterial() (bool, error) {

	appPath := p.appData.AppPath
	repoUrl := p.appData.AppSourceRepoUrl
	chart := p.appData.Chart
	targetRevision := p.appData.AppSourceRevision

	mkDirCmd := "mkdir"
	_, err := utils.CmdExec(mkDirCmd, "", appPath)
	helmChartUrl := fmt.Sprintf("%s/%s-%s.tgz", repoUrl, chart, targetRevision)
	log.Info("[INFO]: appPath: ", appPath)
	output := fmt.Sprintf("%s/%s-%s.tgz", appPath, chart, targetRevision)
	curlCmd := "curl"
	_, err = utils.CmdExec(curlCmd, appPath, helmChartUrl, "--output", output)
	if err != nil {
		log.Infof("[INFO]: Curl Helm Chart CmdExec download : %s ", err.Error())
		return false, err
	}

	helmChartProvUrl := fmt.Sprintf("%s/%s-%s.tgz.prov", repoUrl, chart, targetRevision)
	output = fmt.Sprintf("%s/%s-%s.tgz.prov", appPath, chart, targetRevision)
	_, err = utils.CmdExec(curlCmd, appPath, helmChartProvUrl, "--output", output)
	if err != nil {
		log.Infof("[INFO]: Curl Helm Chart Prov CmdExec download : %s ", err.Error())
		return false, err
	}

	helmCmd := "helm"
	output = fmt.Sprintf("%s/%s-%s.tgz", appPath, chart, targetRevision)
	_, err = utils.CmdExec(helmCmd, appPath, "sigstore", "verify", output)
	if err != nil {
		log.Infof("[INFO]: Helm Sigstore verify CmdExec add : %s ", err.Error())
		return false, err
	}

	log.Info("[INFO]: Helm Sigstore verify was successfull")

	return true, nil

}
