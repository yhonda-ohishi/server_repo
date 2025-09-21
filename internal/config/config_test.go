package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults
	assert.Equal(t, "single", cfg.Deployment.Mode)
	assert.Equal(t, 8080, cfg.Server.HTTPPort)
	assert.Equal(t, 9090, cfg.Server.GRPCPort)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoadFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("DEPLOYMENT_MODE", "separate")
	os.Setenv("SERVER_HTTP_PORT", "8081")
	os.Setenv("SERVER_GRPC_PORT", "9091")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("DEPLOYMENT_MODE")
		os.Unsetenv("SERVER_HTTP_PORT")
		os.Unsetenv("SERVER_GRPC_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "separate", cfg.Deployment.Mode)
	assert.Equal(t, 8081, cfg.Server.HTTPPort)
	assert.Equal(t, 9091, cfg.Server.GRPCPort)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid single mode",
			cfg: &Config{
				Deployment: DeploymentConfig{Mode: "single"},
				Server:     ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
			},
			wantErr: false,
		},
		{
			name: "valid separate mode",
			cfg: &Config{
				Deployment: DeploymentConfig{Mode: "separate"},
				Server:     ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
			},
			wantErr: false,
		},
		{
			name: "invalid deployment mode",
			cfg: &Config{
				Deployment: DeploymentConfig{Mode: "invalid"},
				Server:     ServerConfig{HTTPPort: 8080, GRPCPort: 9090},
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP port",
			cfg: &Config{
				Deployment: DeploymentConfig{Mode: "single"},
				Server:     ServerConfig{HTTPPort: -1, GRPCPort: 9090},
			},
			wantErr: true,
		},
		{
			name: "invalid gRPC port",
			cfg: &Config{
				Deployment: DeploymentConfig{Mode: "single"},
				Server:     ServerConfig{HTTPPort: 8080, GRPCPort: 70000},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModeHelpers(t *testing.T) {
	cfg := &Config{
		Deployment: DeploymentConfig{Mode: "single"},
	}
	assert.True(t, cfg.IsSingleMode())
	assert.False(t, cfg.IsSeparateMode())

	cfg.Deployment.Mode = "separate"
	assert.False(t, cfg.IsSingleMode())
	assert.True(t, cfg.IsSeparateMode())
}