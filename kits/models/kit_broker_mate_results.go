package models

type MateStatus string

const (
	Active  MateStatus = "ACTIVE"
	Standby MateStatus = "STANDBY"
)

type MateResult struct {
	Kit    string
	DC     string
	Mate   string
	Host   string
	Port   int
	Status MateStatus
}

type VPNResult struct {
	Kit string
	DC  string
	VPN string
}
