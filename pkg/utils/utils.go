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

package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	CONFIG_FILE_NAME          = "configmap.yaml"
	MANIFEST_FILE_NAME        = "manifest.yaml"
	SIGNED_MANIFEST_FILE_NAME = "manifest.signed"
	PROVENANCE_FILE_NAME      = "provenance.yaml"
	ATTESTATION_FILE_NAME     = "attestation.json"
	TMP_DIR                   = "/tmp/output"
	PRIVATE_KEY_PATH          = "/etc/signing-secrets/cosign.key"
	PUB_KEY_PATH              = "/etc/signing-secrets/cosign.pub"
)

//GetClient returns a kubernetes client
func GetClient(configpath string) (*kubernetes.Clientset, *rest.Config, error) {

	if configpath == "" {
		log.Debug("Using Incluster configuration")

		config, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("Error occured while reading incluster kubeconfig %s", err.Error())
			return nil, nil, err
		}
		clientset, _ := kubernetes.NewForConfig(config)
		return clientset, config, nil
	}

	config, err := clientcmd.BuildConfigFromFlags("", configpath)
	if err != nil {
		log.Errorf("Error occured while reading kubeconfig %s ", err.Error())
		return nil, nil, err
	}
	clientset, _ := kubernetes.NewForConfig(config)
	return clientset, config, nil
}

func WriteToFile(str, dirPath, filename string) error {

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, os.ModePerm)
	}

	absFilePath := filepath.Join(dirPath, filename)

	f, err := os.Create(absFilePath)
	if err != nil {
		log.Errorf("Error occured while opening file %s ", err.Error())
		return err
	}

	defer f.Close()
	_, err = f.WriteString(str)
	if err != nil {
		log.Errorf("Error occured while writing to file %s ", err.Error())
		return err
	}

	return nil

}

func QueryAPI(url, requestType string, data map[string]interface{}) (string, error) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	token := os.Getenv("ARGOCD_TOKEN")
	var bearer = fmt.Sprintf("Bearer %s", token)
	var dataJson []byte
	if data != nil {
		dataJson, _ = json.Marshal(data)
	} else {
		dataJson = nil
	}
	req, err := http.NewRequest(requestType, url, bytes.NewBuffer(dataJson))
	if err != nil {
		log.Errorf("Error %s ", err.Error())
		return "", err
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error %s", err.Error())
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error %s ", err.Error())
		return "", err
	}

	return string([]byte(body)), nil
}

func RetriveDesiredManifest(appName string) (string, error) {

	baseUrl := os.Getenv("ARGOCD_API_BASE_URL")

	if baseUrl == "" {
		return "", fmt.Errorf("ARGOCD_API_BASE_URL is empty, please specify it in configuration!")
	}

	desiredRscUrl := fmt.Sprintf("%s/%s/managed-resources", baseUrl, appName)

	desiredManifest, err := QueryAPI(desiredRscUrl, "GET", nil)

	if err != nil {
		log.Errorf("Error occured while writing to file %s ", err.Error())
		return "", err
	}

	return desiredManifest, nil
}
