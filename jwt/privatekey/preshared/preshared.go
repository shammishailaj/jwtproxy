// Copyright 2015 CoreOS, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package preshared

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/coreos-inc/jwtproxy/config"
	"github.com/coreos-inc/jwtproxy/jwt/privatekey"
	"github.com/coreos/go-oidc/key"
)

func init() {
	privatekey.Register("preshared", constructor)
}

type Preshared struct {
	*key.PrivateKey
}

type Config struct {
	KeyID          string `yaml:"key_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

func constructor(registrableComponentConfig config.RegistrableComponentConfig) (privatekey.PrivateKey, error) {
	var cfg Config
	bytes, err := yaml.Marshal(registrableComponentConfig.Options)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	return &Preshared{
		PrivateKey: &key.PrivateKey{
			KeyID:      cfg.KeyID,
			PrivateKey: privateKey,
		},
	}, nil
}

func (preshared *Preshared) GetPrivateKey() (*key.PrivateKey, error) {
	return preshared.PrivateKey, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	privateKeyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privateKeyBlock, _ := pem.Decode(privateKeyData)
	if privateKeyBlock == nil {
		return nil, errors.New("bad private key data")
	}

	if privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("unknown key type : %s", privateKeyBlock.Type)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	privateKey.Precompute()

	if err := privateKey.Validate(); err != nil {
		return nil, err
	}

	return privateKey, err
}