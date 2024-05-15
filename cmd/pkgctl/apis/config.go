/*
Copyright 2024 Nokia.

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

package apis

const (
	ViperTokenKey    = "token"
	ViperUsernameKey = "username"
)

type Config struct {
	Secrets map[string]Secret `yaml:"secrets"`
	Repos   map[string]Repo   `yaml:"repos"`
}

type Secret struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Repo struct {
	URL        string `yaml:"url"`
	Deployment bool   `yaml:"deployment"`
	Secret     string `yaml:"secret"`
	Directory  string `yaml:"directory"`
}


