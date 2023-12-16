package migrate

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func V141DeprecatedConfigRename() error {
	var err error
	var confp = filepath.Clean("./data/default/serve.yaml")

	if _, err = os.Stat(confp); err != nil {
		// No renaming is needed if config hasn't been created.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	content, err := os.ReadFile(confp)
	if err != nil {
		return err
	}

	// TODO: Rename fields without reading the whole config file?
	var data map[string]any
	err = yaml.Unmarshal(content, &data)
	if err != nil {
		return err
	}

	if rate, ok := data["customReplenishRate"]; ok {
		data["personalReplenishRate"] = rate
		delete(data, "customReplenishRate")
	}
	if burst, ok := data["customBurst"]; ok {
		data["personalBurst"] = burst
		delete(data, "customBurst")
	}

	modified, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	err = os.WriteFile(confp, modified, 0644)
	if err != nil {
		return err
	}

	return nil
}
