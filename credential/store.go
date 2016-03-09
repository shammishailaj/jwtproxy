// Copyright 2016 CoreOS, Inc
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

package credential

import (
	"fmt"

	"github.com/coreos-inc/hmacproxy/config"
)

type CredentialStoreConstructor func(*config.CredentialSourceConfig) (CredentialStore, error)

var storeFactories = make(map[string]CredentialStoreConstructor)

// RegisterNotifier makes a Fetcher available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func RegisterCredentialStoreFacory(name string, csf func(*config.CredentialSourceConfig) (CredentialStore, error)) {
	if name == "" {
		panic("credentials: could not register a CredentialStore with an empty name")
	}

	if csf == nil {
		panic("credentials: could not register a nil CredentialStore")
	}

	if _, dup := storeFactories[name]; dup {
		panic("credentials: RegisterCredentialStore called twice for " + name)
	}

	storeFactories[name] = csf
}

func CreateCredentialStore(cfg *config.CredentialSourceConfig) (cs CredentialStore, err error) {
	constructor, found := storeFactories[cfg.Type]
	if !found {
		err = fmt.Errorf("credentials: Unable to find credential store constructor for %s", cfg.Type)
		return
	}

	cs, err = constructor(cfg)
	return
}

type CredentialStore interface {
	LoadCredential(keyID, serviceName, regionName string) (*Credential, error)
}