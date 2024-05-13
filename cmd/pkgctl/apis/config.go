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


