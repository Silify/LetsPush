// Package fcm sends notifications through Firebase Cloud Messaging's HTTP v1 API.
package fcm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

// Credentials are the parts of a Firebase service-account key that sending needs.
type Credentials struct {
	ProjectID   string
	ClientEmail string
	privateKey  *rsa.PrivateKey
}

// ParseCredentials reads a service-account key file.
func ParseCredentials(raw []byte) (Credentials, error) {
	var file struct {
		Type        string `json:"type"`
		ProjectID   string `json:"project_id"`
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return Credentials{}, errors.New("the file is not valid JSON")
	}
	if file.Type != "service_account" {
		return Credentials{}, errors.New(`this is not a service-account key (its "type" should be "service_account")`)
	}
	for field, value := range map[string]string{"project_id": file.ProjectID, "client_email": file.ClientEmail, "private_key": file.PrivateKey} {
		if value == "" {
			return Credentials{}, fmt.Errorf("the key has no %s", field)
		}
	}

	block, _ := pem.Decode([]byte(file.PrivateKey))
	if block == nil {
		return Credentials{}, errors.New("the private_key is not in PEM format")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return Credentials{}, errors.New("the private_key could not be read")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return Credentials{}, errors.New("the private_key is not an RSA key")
	}

	return Credentials{ProjectID: file.ProjectID, ClientEmail: file.ClientEmail, privateKey: key}, nil
}
