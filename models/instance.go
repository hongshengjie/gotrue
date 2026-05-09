package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	crudinstances "github.com/netlify/gotrue/crud/instances"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

// Instance represents a GoTrue tenant.
type Instance struct {
	ID         int64               `json:"id"`
	UUID       string              `json:"uuid,omitempty"`
	BaseConfig *conf.Configuration `json:"config"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

// toInstancesCrud converts a models.Instance to a crud Instances struct.
func toInstancesCrud(i *Instance) (*crudinstances.Instances, error) {
	var rawConfig string
	if i.BaseConfig != nil {
		data, err := json.Marshal(i.BaseConfig)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling instance config")
		}
		rawConfig = string(data)
	} else {
		rawConfig = "{}"
	}
	return &crudinstances.Instances{
		Id:            i.ID,
		Uuid:          i.UUID,
		RawBaseConfig: rawConfig,
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
	}, nil
}

// fromInstancesCrud converts a crud Instances to a models.Instance.
func fromInstancesCrud(ci *crudinstances.Instances) (*Instance, error) {
	i := &Instance{
		ID:        ci.Id,
		UUID:      ci.Uuid,
		CreatedAt: ci.CreatedAt,
		UpdatedAt: ci.UpdatedAt,
	}
	if ci.RawBaseConfig != "" && ci.RawBaseConfig != "{}" {
		cfg := &conf.Configuration{}
		if err := json.Unmarshal([]byte(ci.RawBaseConfig), cfg); err != nil {
			return nil, errors.Wrap(err, "unmarshalling instance config")
		}
		i.BaseConfig = cfg
	}
	return i, nil
}

// Config loads the base configuration values with defaults applied.
func (i *Instance) Config() (*conf.Configuration, error) {
	if i.BaseConfig == nil {
		return nil, errors.New("no configuration data available")
	}
	baseConf := &conf.Configuration{}
	*baseConf = *i.BaseConfig
	baseConf.ApplyDefaults()
	return baseConf, nil
}

// UpdateConfig updates the base configuration for this instance.
func (i *Instance) UpdateConfig(tx *storage.Connection, config *conf.Configuration) error {
	i.BaseConfig = config
	data, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshalling instance config")
	}
	ctx := context.Background()
	_, err = crudinstances.Update(tx.DB()).
		SetRawBaseConfig(string(data)).
		Where(crudinstances.IdOp.EQ(i.ID)).
		Save(ctx)
	return errors.Wrap(err, "error updating instance config")
}

// GetInstance finds an instance by its internal bigint ID.
func GetInstance(tx *storage.Connection, instanceID int64) (*Instance, error) {
	ctx := context.Background()
	ci, err := crudinstances.Find(tx.DB()).Where(crudinstances.IdOp.EQ(instanceID)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return fromInstancesCrud(ci)
}

// GetInstanceByUUID finds an instance by its external UUID string.
func GetInstanceByUUID(tx *storage.Connection, uuidStr string) (*Instance, error) {
	ctx := context.Background()
	ci, err := crudinstances.Find(tx.DB()).Where(crudinstances.UuidOp.EQ(uuidStr)).One(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return fromInstancesCrud(ci)
}

// CreateInstance persists a new instance to the database.
func CreateInstance(tx *storage.Connection, i *Instance) error {
	now := time.Now()
	if i.CreatedAt.IsZero() {
		i.CreatedAt = now
	}
	if i.UpdatedAt.IsZero() {
		i.UpdatedAt = now
	}
	ci, err := toInstancesCrud(i)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = crudinstances.Create(tx.DB()).SetInstances(ci).Save(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating instance")
	}
	// Set the auto-increment ID back on the model.
	i.ID = ci.Id
	return nil
}

// DeleteInstance deletes an instance and all related data in a transaction.
func DeleteInstance(conn *storage.Connection, instance *Instance) error {
	return conn.Transaction(func(tx *storage.Connection) error {
		ctx := context.Background()
		if err := tx.Exec("DELETE FROM users WHERE instance_id = ?", instance.ID); err != nil {
			return errors.Wrap(err, "error deleting user records")
		}
		if err := tx.Exec("DELETE FROM refresh_tokens WHERE instance_id = ?", instance.ID); err != nil {
			return errors.Wrap(err, "error deleting refresh token records")
		}
		_, err := crudinstances.Delete(tx.DB()).Where(crudinstances.IdOp.EQ(instance.ID)).Exec(ctx)
		return errors.Wrap(err, "error deleting instance record")
	})
}
