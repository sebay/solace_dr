package models

type KitsFile struct {
	Kits map[string]Kit `yaml:"kits"`
}

type Kit struct {
	DC1 DC `yaml:"dc1"`
	DC2 DC `yaml:"dc2"`
}

type DC struct {
	Mate1 Endpoint `yaml:"mate1"`
	Mate2 Endpoint `yaml:"mate2"`
}

type Endpoint struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AboutResult struct {
	Kit         string `json:"kit"`
	DC          string `json:"dc"`
	Mate        string `json:"mate"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Platform    string `json:"platform"`
	SempVersion string `json:"sempVersion"`
	Release     string `json:"releaseVersion"`
	ApiVersion  string `json:"apiVersion"`
	Build       string `json:"buildVersion,omitempty"`
	Description string `json:"description,omitempty"`
}
