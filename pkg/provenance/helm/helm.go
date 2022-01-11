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
	"fmt"

	"github.com/IBM/argocd-interlace/pkg/application"
	"github.com/IBM/argocd-interlace/pkg/utils"
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

func (p Provenance) GenerateProvanance(target, targetDigest string, uploadTLog bool) error {
	return nil
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
