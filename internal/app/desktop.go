package app

import (
	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/service"
	"io"
)

func desktopConfiguration(home string, out io.Writer) error {
	c, e := config.Load(home)
	if e != nil {
		return e
	}
	exe, e := service.Executable()
	if e != nil {
		return e
	}
	return printJSON(out, map[string]any{"inferenceProvider": "gateway", "inferenceGatewayBaseUrl": "http://" + c.Listen + "/claude", "inferenceGatewayAuthScheme": "bearer", "inferenceCredentialKind": "helper-script", "inferenceCredentialHelper": exe, "inferenceCredentialHelperArgs": []string{"auth-token", "--home", home}})
}
