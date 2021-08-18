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

package provenance

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gajananan/argocd-interlace/pkg/utils"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/pkg/ssl"
	"github.com/sigstore/cosign/pkg/cosign"
	log "github.com/sirupsen/logrus"
	"github.com/theupdateframework/go-tuf/encrypted"
	"golang.org/x/term"
)

type IntotoSigner struct {
	priv *ecdsa.PrivateKey
}

const (
	cli         = "/usr/local/bin/rekor-cli"
	server      = "../rekor-server"
	nodeDataDir = "node"
)

type SignOpts struct {
	Pf cosign.PassFunc
}

var (
	// Read is for fuzzing
	Read = readPasswordFn
)

func GenerateProvanance(appName, appPath,
	appSourceRepoUrl, appSourceRevision, appSourceCommitSha,
	imageRef string, buildStartedOn, buildFinishedOn time.Time) error {

	subjects := []in_toto.Subject{}
	productName := imageRef

	digest, err := getDigest(productName)
	if err != nil {
		log.Errorf("Error in getting digest: %s ", err.Error())
		return err
	}

	digest = strings.ReplaceAll(digest, "sha256:", "")
	log.Info("digest ", digest)
	subjects = append(subjects, in_toto.Subject{Name: productName,
		Digest: in_toto.DigestSet{
			"sha256": digest,
		},
	})

	materials := generateMaterial(appName, appPath, appSourceRepoUrl, appSourceRevision, appSourceCommitSha)

	entryPoint := "argocd-interlace"
	recipe := in_toto.ProvenanceRecipe{
		EntryPoint: entryPoint,
		Arguments:  []string{"-n argocd"},
	}

	it := in_toto.Statement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: in_toto.PredicateProvenanceV01,
			Subject:       subjects,
		},
		Predicate: in_toto.ProvenancePredicate{
			Metadata: in_toto.ProvenanceMetadata{
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

	appDirPath := filepath.Join(utils.TMP_DIR, appName, appPath)

	err = utils.WriteToFile(string(b), appDirPath, utils.PROVENANCE_FILE_NAME)
	if err != nil {
		log.Errorf("Error in writing provenance to a file:  %s", err.Error())
		return err
	}

	generateSignedAttestation(it, appDirPath)

	return nil
}

func getDigest(src string) (string, error) {

	digest, err := crane.Digest(src)
	if err != nil {
		return "", fmt.Errorf("fetching digest %s: %v", src, err)
	}
	return digest, nil
}

func generateMaterial(appName, appPath, appSourceRepoUrl, appSourceRevision, appSourceCommitSha string) []in_toto.ProvenanceMaterial {

	materials := []in_toto.ProvenanceMaterial{}

	materials = append(materials, in_toto.ProvenanceMaterial{
		URI: appSourceRepoUrl,
		Digest: in_toto.DigestSet{
			"commit":   string(appSourceCommitSha),
			"revision": appSourceRevision,
			"path":     appPath,
		},
	})

	return materials
}

func generateSignedAttestation(it in_toto.Statement, appDirPath string) error {

	b, err := json.Marshal(it)
	if err != nil {
		log.Errorf("Error in marshaling attestation:  %s", err.Error())
		return err
	}

	ecdsaPriv, err := ioutil.ReadFile(filepath.Clean(utils.PRIVATE_KEY_PATH))
	if err != nil {
		log.Errorf("Error in reading private key:  %s", err.Error())
		return err
	}

	pb, _ := pem.Decode(ecdsaPriv)

	pwd := "" //os.Getenv(cosignPwd) //GetPass(true)

	x509Encoded, err := encrypted.Decrypt(pb.Bytes, []byte(pwd))

	if err != nil {
		log.Errorf("Error in dycrypting private key: %s", err.Error())
		return err
	}
	priv, err := x509.ParsePKCS8PrivateKey(x509Encoded)

	if err != nil {
		log.Errorf("Error in parsing private key: %s", err.Error())
		return err
	}

	signer, err := ssl.NewEnvelopeSigner(&IntotoSigner{
		priv: priv.(*ecdsa.PrivateKey),
	})
	if err != nil {
		log.Errorf("Error in creating new signer: %s", err.Error())
		return err
	}

	env, err := signer.SignPayload("application/vnd.in-toto+json", b)
	if err != nil {
		log.Errorf("Error in signing payload: %s", err.Error())
		return err
	}

	// Now verify
	err = signer.Verify(env)
	if err != nil {
		log.Errorf("Error in verifying env: %s", err.Error())
		return err
	}

	eb, err := json.Marshal(env)
	if err != nil {
		log.Errorf("Error in marshaling env: %s", err.Error())
		return err
	}

	log.Debug("attestation.json", string(eb))

	err = utils.WriteToFile(string(eb), appDirPath, utils.ATTESTATION_FILE_NAME)
	if err != nil {
		log.Errorf("Error in writing attestation to a file: %s", err.Error())
		return err
	}

	attestationPath := filepath.Join(appDirPath, utils.ATTESTATION_FILE_NAME)

	upload(it, attestationPath)

	return nil

}

func readPasswordFn() func() ([]byte, error) {
	pw, ok := os.LookupEnv("COSIGN_PASSWORD")
	switch {
	case ok:
		return func() ([]byte, error) {
			return []byte(pw), nil
		}
	case term.IsTerminal(0):
		return func() ([]byte, error) {
			return term.ReadPassword(0)
		}
	// Handle piped in passwords.
	default:
		return func() ([]byte, error) {
			return ioutil.ReadAll(os.Stdin)
		}
	}
}

func GetPass(confirm bool) ([]byte, error) {
	read := Read()
	fmt.Fprint(os.Stderr, "Enter password for private key: ")
	pw1, err := read()
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if !confirm {
		return pw1, nil
	}
	fmt.Fprint(os.Stderr, "Enter again: ")
	pw2, err := read()
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}

	if string(pw1) != string(pw2) {
		return nil, errors.New("passwords do not match")
	}
	return pw1, nil
}

func (it *IntotoSigner) Sign(data []byte) ([]byte, string, error) {
	h := sha256.Sum256(data)
	sig, err := it.priv.Sign(rand.Reader, h[:], crypto.SHA256)
	if err != nil {
		return nil, "", err
	}
	return sig, "", nil
}

func (it *IntotoSigner) Verify(_ string, data, sig []byte) error {
	h := sha256.Sum256(data)
	ok := ecdsa.VerifyASN1(&it.priv.PublicKey, h[:], sig)
	if ok {
		return nil
	}
	return errors.New("invalid signature")
}

func upload(it in_toto.Statement, attestationPath string) {

	pubKeyPath := utils.PUB_KEY_PATH
	// If we do it twice, it should already exist
	out := runCli("upload", "--artifact", attestationPath, "--type", "intoto", "--public-key", pubKeyPath, "--pki-format", "x509")

	outputContains(out, "Created entry at")

	uuid := getUUIDFromUploadOutput(out)

	log.Infof("Uploaded attestation to tlog,  uuid: %s", uuid)
}

func outputContains(output, sub string) {

	if !strings.Contains(output, sub) {
		log.Infof(fmt.Sprintf("Expected [%s] in response, got %s", sub, output))
	}
}

func getUUIDFromUploadOutput(out string) string {

	// Output looks like "Artifact timestamped at ...\m Wrote response \n Created entry at index X, available at $URL/UUID", so grab the UUID:
	urlTokens := strings.Split(strings.TrimSpace(out), " ")
	url := urlTokens[len(urlTokens)-1]
	splitUrl := strings.Split(url, "/")
	return splitUrl[len(splitUrl)-1]
}

func runCli(arg ...string) string {
	rekorServer := os.Getenv("REKOR_SERVER")

	argStr := fmt.Sprintf("--rekor_server=%s", rekorServer)

	arg = append(arg, argStr)
	// use a blank config file to ensure no collision
	if os.Getenv("REKORTMPDIR") != "" {
		arg = append(arg, "--config="+os.Getenv("REKORTMPDIR")+".rekor.yaml")
	}
	return run("", cli, arg...)

}

func run(stdin, cmd string, arg ...string) string {

	c := exec.Command(cmd, arg...)
	if stdin != "" {
		c.Stdin = strings.NewReader(stdin)
	}
	if os.Getenv("REKORTMPDIR") != "" {
		// ensure that we use a clean state.json file for each run
		c.Env = append(c.Env, "HOME="+os.Getenv("REKORTMPDIR"))
	}
	b, err := c.CombinedOutput()
	if err != nil {
		log.Infof("Error in executing CLI: %s", string(b))
	}
	return string(b)
}
