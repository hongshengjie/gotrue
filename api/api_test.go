package api

import (
	"context"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
)

const (
	apiTestVersion = "1"
	apiTestConfig  = "../hack/test.env"
)

// setupAPIForTest creates a new API to run tests with.
func setupAPIForTest() (*API, *conf.Configuration, error) {
	return setupAPIForTestWithCallback(nil)
}

func setupAPIForMultiinstanceTest() (*API, *conf.Configuration, error) {
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn *storage.Connection) (int64, error) {
		gc.MultiInstanceMode = true
		return 0, nil
	}

	return setupAPIForTestWithCallback(cb)
}

func setupAPIForTestForInstance() (*API, *conf.Configuration, int64, error) {
	var instanceID int64
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn *storage.Connection) (int64, error) {
		i := &models.Instance{
			UUID:       testUUID,
			BaseConfig: c,
		}
		if err := models.CreateInstance(conn, i); err != nil {
			return 0, err
		}
		instanceID = i.ID
		return i.ID, nil
	}

	api, cfg, err := setupAPIForTestWithCallback(cb)
	if err != nil {
		return nil, nil, 0, err
	}
	return api, cfg, instanceID, nil
}

func setupAPIForTestWithCallback(cb func(*conf.GlobalConfiguration, *conf.Configuration, *storage.Connection) (int64, error)) (*API, *conf.Configuration, error) {
	globalConfig, err := conf.LoadGlobal(apiTestConfig)
	if err != nil {
		return nil, nil, err
	}

	conn, err := test.SetupDBConnection(globalConfig)
	if err != nil {
		return nil, nil, err
	}

	config, err := conf.LoadConfig(apiTestConfig)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	instanceID := int64(0)
	if cb != nil {
		instanceID, err = cb(globalConfig, config, conn)
		if err != nil {
			conn.Close()
			return nil, nil, err
		}
	}

	ctx, err := WithInstanceConfig(context.Background(), config, instanceID)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return NewAPIWithVersion(ctx, globalConfig, conn, apiTestVersion), config, nil
}

func TestEmailEnabledByDefault(t *testing.T) {
	api, _, err := setupAPIForTest()
	require.NoError(t, err)

	require.False(t, api.config.External.Email.Disabled)
}
