package activities

import (
	"kits-worker/kits/models"
	"net/http"
)

func applyBasicAuth(req *http.Request, auth models.BasicAuth) {
	req.SetBasicAuth(auth.Username, auth.Password)
}
