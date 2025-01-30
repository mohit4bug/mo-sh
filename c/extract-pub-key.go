package c

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func ExtractPublicKey(privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	var privateKey interface{}
	var err error

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		err = fmt.Errorf("unsupported key type: %s", block.Type)
	}
	if err != nil {
		return "", err
	}

	var sshPublicKey ssh.PublicKey
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		sshPublicKey, err = ssh.NewPublicKey(&key.PublicKey)
	case *ecdsa.PrivateKey:
		sshPublicKey, err = ssh.NewPublicKey(&key.PublicKey)
	case ed25519.PrivateKey:
		sshPublicKey, err = ssh.NewPublicKey(key.Public().(ed25519.PublicKey))
	default:
		err = fmt.Errorf("unsupported key type")
	}
	if err != nil {
		return "", err
	}

	return string(ssh.MarshalAuthorizedKey(sshPublicKey)), nil
}
