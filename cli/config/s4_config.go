package config

type S4RemoteConfig struct {
	RemoteURL string       `json:"remote_url"`
	Auth      *S4BasicAuth `json:"auth"`
}

func (cfg *S4RemoteConfig) URL() string {
	return cfg.RemoteURL
}

func (cfg *S4RemoteConfig) Type() RemoteType {
	return RemoteTypeS4
}

type S4BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
