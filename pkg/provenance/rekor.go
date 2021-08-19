package provenance

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ibm/argocd-interlace/pkg/utils"
	"github.com/sigstore/cosign/pkg/cosign"
	rc "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	log "github.com/sirupsen/logrus"
)

func uploadTlog(payloadFormat, payloadPath string) {

	payload, err := ioutil.ReadFile(payloadPath)

	rawPayload, err := json.Marshal(payload)

	if err != nil {
		log.Warnf("Unable to marshal payload: %s", err.Error())
	}

	privateKey, err := ioutil.ReadFile(filepath.Clean(utils.PRIVATE_KEY_PATH))

	signer, err := cosign.LoadECDSAPrivateKey(privateKey, []byte(""))

	signature, err := signer.SignMessage(bytes.NewReader(rawPayload))

	pub, err := signer.PublicKey()

	if err != nil {
		log.Errorf("getting public key %s", err.Error())
	}
	pem, err := cryptoutils.MarshalPublicKeyToPEM(pub)

	if err != nil {
		log.Errorf("key to pem %s", err.Error())

	}

	rekorServerUrl := os.Getenv("REKOR_SERVER")
	rekorCliet, err := rc.GetRekorClient(rekorServerUrl)

	if err != nil {
		log.Errorf("Error in getting rekor client: %s", err.Error())
	}

	if payloadFormat == "in-toto" {
		logEntryAnon, err := cosign.UploadAttestationTLog(rekorCliet, signature, pem)
		if err != nil {
			log.Errorf("Error in uploading attestation to tlog: %s", err.Error())
		}
		log.Info("logEntryAnon ", logEntryAnon.LogIndex)
	}

}
