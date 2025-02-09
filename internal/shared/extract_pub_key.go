package shared

import (
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func ExtractPublicKey(privateKeyPEM string, keyType string) (string, error) {
	if keyType != "rsa" && keyType != "ed25519" {
		return "", fmt.Errorf("unsupported key type")
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	var signer ssh.Signer
	var err error

	signer, err = ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return "", err
	}

	pubKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))

	if !strings.HasPrefix(pubKey, "ssh-"+keyType) {
		return "", fmt.Errorf("key type mismatch")
	}

	return pubKey, nil
}
