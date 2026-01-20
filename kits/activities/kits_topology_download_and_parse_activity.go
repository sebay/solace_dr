package activities

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"kits-worker/kits/models"
	"net/http"
	"strings"

	"go.temporal.io/sdk/activity"
	"gopkg.in/yaml.v3"
)

func DownloadAndParseKitsActivity(
	ctx context.Context,
	url string,
	filter string,
) (map[string]models.Kit, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create gzip reader
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var kitsYAML []byte

	// Iterate through files in the tarball
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		// Match kits.yaml (allow nested paths)
		if hdr.Typeflag == tar.TypeReg && strings.HasSuffix(hdr.Name, "kits.yaml") {
			kitsYAML, err = io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read kits.yaml: %w", err)
			}
			break
		}
	}

	if kitsYAML == nil {
		return nil, fmt.Errorf("kits.yaml not found in archive")
	}

	var file models.KitsFile
	if err := yaml.Unmarshal(kitsYAML, &file); err != nil {
		return nil, err
	}

	result := make(map[string]models.Kit)
	for name, kit := range file.Kits {
		if filter != "" && !strings.Contains(name, filter) {
			continue
		}
		result[name] = kit
	}

	activity.GetLogger(ctx).Info("kits parsed from tgz", "count", len(result))
	return result, nil
}
