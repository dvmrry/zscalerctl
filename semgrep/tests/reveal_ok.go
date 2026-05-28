//go:build semgrepfixtures

package tests

import "github.com/dvmrry/zscalerctl/internal/secret"

type sdkConfiguration struct {
	ClientID     string
	ClientSecret string
}

type readerConfig struct {
	ClientID     secret.Secret
	ClientSecret secret.Secret
}

func newSDKConfiguration(cfg readerConfig) *sdkConfiguration {
	return &sdkConfiguration{
		ClientID:     cfg.ClientID.Reveal(),
		ClientSecret: cfg.ClientSecret.Reveal(),
	}
}
