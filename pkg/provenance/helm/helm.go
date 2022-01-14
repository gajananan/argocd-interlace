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
	"github.com/IBM/argocd-interlace/pkg/provenance/attestation"
	"github.com/IBM/argocd-interlace/pkg/utils"
	"github.com/in-toto/in-toto-golang/in_toto"
	log "github.com/sirupsen/logrus"
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
	appSourceRevision := p.appData.AppSourceRevision
	appDirPath := p.appData.AppDirPath
	chart := p.appData.Chart

	entryPoint := "helm install"
	helmChart := fmt.Sprintf("%s-%s.tgz", chart, appSourceRevision)
	recipe := in_toto.ProvenanceRecipe{
		EntryPoint: entryPoint,
		Arguments:  []string{chart + "  " + helmChart},
	}

	subjects := []in_toto.Subject{}

	targetDigest = strings.ReplaceAll(targetDigest, "sha256:", "")
	subjects = append(subjects, in_toto.Subject{Name: target,
		Digest: in_toto.DigestSet{
			"sha256": targetDigest,
		},
	})

	materials := p.generateMaterial()

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

func (p Provenance) generateMaterial() []in_toto.ProvenanceMaterial {

	appPath := p.appData.AppPath
	appSourceRepoUrl := p.appData.AppSourceRepoUrl
	appSourceRevision := p.appData.AppSourceRevision
	chart := p.appData.Chart
	values := p.appData.Values
	materials := []in_toto.ProvenanceMaterial{}

	helmChartPath := fmt.Sprintf("%s/%s-%s.tgz", appPath, chart, appSourceRevision)
	chartHash, _ := utils.ComputeHash(helmChartPath)

	materials = append(materials, in_toto.ProvenanceMaterial{
		URI: appSourceRepoUrl + ".git",
		Digest: in_toto.DigestSet{
			"sha256hash": chartHash,
			"revision":   appSourceRevision,
			"name":       chart,
		},
	})

	materials = append(materials, in_toto.ProvenanceMaterial{

		Digest: in_toto.DigestSet{
			"material":   "values",
			"parameters": values,
		},
	})
	return materials
}

func (p Provenance) VerifySourceMaterial() (bool, error) {

	appPath := p.appData.AppPath
	repoUrl := p.appData.AppSourceRepoUrl
	chart := p.appData.Chart
	targetRevision := p.appData.AppSourceRevision

	mkDirCmd := "mkdir"
	_, err := utils.CmdExec(mkDirCmd, "", appPath)
	helmChartUrl := fmt.Sprintf("%s/%s-%s.tgz", repoUrl, chart, targetRevision)

	chartPath := fmt.Sprintf("%s/%s-%s.tgz", appPath, chart, targetRevision)
	curlCmd := "curl"
	_, err = utils.CmdExec(curlCmd, appPath, helmChartUrl, "--output", chartPath)
	if err != nil {
		log.Infof("Retrive Helm Chart : %s ", err.Error())
		return false, err
	}

	helmChartProvUrl := fmt.Sprintf("%s/%s-%s.tgz.prov", repoUrl, chart, targetRevision)
	provPath := fmt.Sprintf("%s/%s-%s.tgz.prov", appPath, chart, targetRevision)
	_, err = utils.CmdExec(curlCmd, appPath, helmChartProvUrl, "--output", provPath)
	if err != nil {
		log.Infof("Retrive Helm Chart Prov : %s ", err.Error())
		return false, err
	}

	helmCmd := "helm"

	_, err = utils.CmdExec(helmCmd, appPath, "sigstore", "verify", chartPath)
	if err != nil {
		log.Infof("Helm-sigstore verify : %s ", err.Error())
		return false, err
	}

	log.Infof("[INFO]: Helm sigstore verify was successful for the  Helm chart: %s ", p.appData.Chart)

	return true, nil

}
