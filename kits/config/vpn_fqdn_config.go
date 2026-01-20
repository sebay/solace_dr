package config

const (
	// todo update this:
	// write proper logic based on actual broker fqdn and also do any cleanup on the fqdn standards accross dev, staging, prod
	VPN_PRIMARY_BROKER_FQDN_GLOBAL_DOMAIN = "-solace-a.local"
	VPN_BACKUP_BROKER_FQDN_GLOBAL_DOMAIN  = "-solace-b.local"

	// ie: vpn-platform-p1-1-a-prd.id.app.domain.com
	//VPN_PRIMARY_BROKER_FQDN_GLOBAL_DOMAIN = "-1-a-prd.id.app.domain.com"
	//VPN_BACKUP_BROKER_FQDN_GLOBAL_DOMAIN = "-1-b-prd.id.app.domain.com"
)
