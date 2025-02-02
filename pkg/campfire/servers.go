/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package campfire

import (
	_ "embed"
	"strings"
	"sync"
)

// defaultTURNServers is a list of default TURN servers gathered from always-online-stun.
var defaultTURNServers []string

//go:embed valid_hosts.txt
var alwaysOnHostsFile []byte

var once sync.Once

// GetDefaultTURNServers returns the default list of TURN servers.
func GetDefaultTURNServers() []string {
	once.Do(func() {
		defaultTURNServers = strings.Split(strings.TrimSpace(string(alwaysOnHostsFile)), "\n")
	})
	return defaultTURNServers
}
