package sshGen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// SSHKey has PublicKey in rsa format and PrivateKey in PEM format
type SSHKey struct {
	PublicKey  string
	PrivateKey string
}

// Gen - generate ssh key pair; TODO refactor
func Gen() (*SSHKey, error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	// this is human readable PEM format ---BEGIN CERTIFICATE--- ...
	privateKeyPem := string(pem.EncodeToMemory(&privateKeyBlock))

	publicKey := privateKey.PublicKey
	publicKeyDer, err := x509.MarshalPKIXPublicKey(&publicKey)

	if err != nil {
		return nil, err
	}

	publicKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}
	// this is human readable PEM format ---BEGIN CERTIFICATE--- ...
	publicKeyPem := string(pem.EncodeToMemory(&publicKeyBlock))

	// convert to RSA format
	block, _ := pem.Decode([]byte(publicKeyPem))
	pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
	sshKey, _ := pub.(*rsa.PublicKey)
	pk, _ := ssh.NewPublicKey(sshKey)
	sshPubKey := base64.StdEncoding.EncodeToString(pk.Marshal())

	return &SSHKey{PublicKey: sshPubKey, PrivateKey: privateKeyPem}, nil

}
