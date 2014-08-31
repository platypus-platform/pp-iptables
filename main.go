package main

import (
	"flag"
	"fmt"
	"github.com/platypus-platform/pp-logging"
	"github.com/platypus-platform/pp-store"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

type IptablesConfig struct {
	PortAuthorityConfig string
	PortAuthorityPath   string
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(1)
	}

	var config IptablesConfig

	flag.StringVar(&config.PortAuthorityConfig, "config-dir",
		"fake/portauthority.d", "config directory for portauthority")
	flag.StringVar(&config.PortAuthorityPath, "cmd",
		"port_authority", "binary for portauthority")
	flag.Parse()

	err = pp.PollIntent(hostname, func(intent pp.IntentNode) {
		for _, app := range intent.Apps {
			if err := configurePortAuthority(config, app); err != nil {
				logger.Fatal("%s: error configuring portauthority: %s", app.Name, err)
			}
		}
		// TODO: Remove old definitions
		logger.Info("refreshing portauthority")
		if err := refreshPortAuthority(config); err != nil {
			logger.Fatal("error refreshing portauthority: %s", err)
		}
	})

	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(1)
	}
}

type PortDefinition struct {
	Port     int
	Protocol string
}

func configurePortAuthority(config IptablesConfig, app pp.IntentApp) error {
	data := map[string]PortDefinition{}

	for _, port := range app.Ports {
		protocol := "tcp"
		key := fmt.Sprintf("%s_%d_%s", app.Name, port, protocol)

		data[key] = PortDefinition{
			Port:     port,
			Protocol: protocol,
		}

		logger.Info("%s: allowing %s:%d", app.Name, protocol, port)
	}

	fpath := path.Join(config.PortAuthorityConfig, app.Name+".yml")
	fcontent, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(fpath, fcontent, 0644); err != nil {
		return err
	}

	return nil
}

func refreshPortAuthority(config IptablesConfig) error {
	cmd := exec.Command(config.PortAuthorityPath, "rebuild")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func writeFileAtomic(fpath string, fcontent []byte, mod os.FileMode) error {
	f, err := ioutil.TempFile("", "pp-iptables")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // This will fail in happy case, that's fine.

	if _, err := f.Write(fcontent); err != nil {
		return err
	}
	if err := f.Chmod(mod); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), fpath); err != nil {
		return err
	}

	return nil
}
