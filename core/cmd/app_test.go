package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/env"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils/configtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

var (
	setInFile = "set in config file"
	setInEnv  = "set in env"

	testEnvContents = fmt.Sprintf("P2P.V2.AnnounceAddresses = ['%s']", setInEnv)

	testConfigFileContents = erinaceus.Config{
		Core: toml.Core{
			RootDir: &setInFile,
		},
	}

	testSecretsFileContents = erinaceus.Secrets{
		Secrets: toml.Secrets{
			Prometheus: toml.PrometheusSecrets{
				AuthToken: models.NewSecret("PROM_TOKEN"),
			},
		},
	}

	testSecretsRedactedContents = erinaceus.Secrets{
		Secrets: toml.Secrets{
			Prometheus: toml.PrometheusSecrets{
				AuthToken: models.NewSecret("xxxxx"),
			},
		},
	}
)

func withDefaults(t *testing.T, c erinaceus.Config, s erinaceus.Secrets) erinaceus.GeneralConfig {
	cfg, err := erinaceus.GeneralConfigOpts{Config: c, Secrets: s}.New()
	require.NoError(t, err)
	return cfg
}

func Test_initServerConfig(t *testing.T) {
	type args struct {
		opts         *erinaceus.GeneralConfigOpts
		fileNames    []string
		secretsFiles []string
		envVar       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantCfg erinaceus.GeneralConfig
	}{
		{
			name: "env only",
			args: args{
				opts:   new(erinaceus.GeneralConfigOpts),
				envVar: testEnvContents,
			},
			wantCfg: withDefaults(t, erinaceus.Config{
				Core: toml.Core{},
			}, erinaceus.Secrets{}),
		},
		{
			name: "files only",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
			},
			wantCfg: withDefaults(t, testConfigFileContents, erinaceus.Secrets{}),
		},
		{
			name: "file error",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{"notexist"},
			},
			wantErr: true,
		},
		{
			name: "env overlay of file",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				envVar:    testEnvContents,
			},
			wantCfg: withDefaults(t, erinaceus.Config{
				Core: toml.Core{
					RootDir: &setInFile,
				},
			}, erinaceus.Secrets{}),
		},
		{
			name: "failed to read secrets",
			args: args{
				opts:         new(erinaceus.GeneralConfigOpts),
				fileNames:    []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{"/doesnt-exist"},
			},
			wantErr: true,
		},
		{
			name: "reading secrets",
			args: args{
				opts:         new(erinaceus.GeneralConfigOpts),
				fileNames:    []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{configtest.WriteTOMLFile(t, testSecretsFileContents, "test_secrets.toml")},
			},
			wantCfg: withDefaults(t, testConfigFileContents, testSecretsRedactedContents),
		},
		{
			name: "reading multiple secrets",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../services/chainlink/testdata/mergingsecretsdata/secrets-database.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-password.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-pyroscope.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-prometheus.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-mercury-split-one.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-mercury-split-two.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-threshold.toml",
					"../services/chainlink/testdata/mergingsecretsdata/secrets-webserver-ldap.toml",
				},
			},
			wantErr: false,
		},
		{
			name: "reading multiple secrets with overrides: Database",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-database.toml",
					"../testdata/mergingsecretsdata/secrets-database.toml",
				},
			},
			wantErr: true,
		},
		{
			name: "reading multiple secrets with overrides: Password",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-password.toml",
					"../testdata/mergingsecretsdata/secrets-password.toml",
				},
			},
			wantErr: true,
		},
		{
			name: "reading multiple secrets with overrides: Pyroscope",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-pyroscope.toml",
					"../testdata/mergingsecretsdata/secrets-pyroscope.toml",
				},
			},
			wantErr: true,
		},
		{
			name: "reading multiple secrets with overrides: Prometheus",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-prometheus.toml",
					"../testdata/mergingsecretsdata/secrets-prometheus.toml",
				},
			},
			wantErr: true,
		},
		{
			name: "reading multiple secrets with overrides: Mercury",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-mercury-split-one.toml",
					"../testdata/mergingsecretsdata/secrets-mercury-split-one.toml",
				},
			},
			wantErr: true,
		},
		{
			name: "reading multiple secrets with overrides: Threshold",
			args: args{
				opts:      new(erinaceus.GeneralConfigOpts),
				fileNames: []string{configtest.WriteTOMLFile(t, testConfigFileContents, "test.toml")},
				secretsFiles: []string{
					"../testdata/mergingsecretsdata/secrets-threshold.toml",
					"../testdata/mergingsecretsdata/secrets-threshold.toml",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envVar != "" {
				t.Setenv(string(env.Config), tt.args.envVar)
			}
			cfg, err := initServerConfig(tt.args.opts, tt.args.fileNames, tt.args.secretsFiles)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadOpts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCfg != nil {
				assert.Equal(t, tt.wantCfg, cfg)
			}
		})
	}
}
