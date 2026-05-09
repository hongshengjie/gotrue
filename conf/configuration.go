package conf

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/BurntSushi/toml"
)

// OAuthProviderConfiguration holds all config related to external account providers.
type OAuthProviderConfiguration struct {
	ClientID    string `json:"client_id" toml:"client_id"`
	Secret      string `json:"secret" toml:"secret"`
	RedirectURI string `json:"redirect_uri" toml:"redirect_uri"`
	URL         string `json:"url" toml:"url"`
	Enabled     bool   `json:"enabled" toml:"enabled"`
}

type EmailProviderConfiguration struct {
	Disabled bool `json:"disabled" toml:"disabled"`
}

type SamlProviderConfiguration struct {
	Enabled     bool   `json:"enabled" toml:"enabled"`
	MetadataURL string `json:"metadata_url" toml:"metadata_url"`
	APIBase     string `json:"api_base" toml:"api_base"`
	Name        string `json:"name" toml:"name"`
	SigningCert string `json:"signing_cert" toml:"signing_cert"`
	SigningKey  string `json:"signing_key" toml:"signing_key"`
}

// DBConfiguration holds all the database related configuration.
type DBConfiguration struct {
	Driver         string `json:"driver" toml:"driver"`
	URL            string `json:"url" toml:"url"`
	Namespace      string `json:"namespace" toml:"namespace"`
	MigrationsPath string `json:"migrations_path" toml:"migrations_path"`
}

// JWTConfiguration holds all the JWT related configuration.
type JWTConfiguration struct {
	Secret               string `json:"secret" toml:"secret"`
	Exp                  int    `json:"exp" toml:"exp"`
	Aud                  string `json:"aud" toml:"aud"`
	AdminGroupName       string `json:"admin_group_name" toml:"admin_group_name"`
	DefaultGroupName     string `json:"default_group_name" toml:"default_group_name"`
	RefreshTokenLifetime int    `json:"refresh_token_lifetime" toml:"refresh_token_lifetime"`
}

// GlobalConfiguration holds all the configuration that applies to all instances.
type GlobalConfiguration struct {
	API struct {
		Host            string
		Port            int
		Endpoint        string
		RequestIDHeader string
		ExportSecret    string
	}
	DB                DBConfiguration
	External          ProviderConfiguration
	Logging           LoggingConfig
	OperatorToken     string
	MultiInstanceMode bool
	Tracing           TracingConfig
	SMTP              SMTPConfiguration
	RateLimitHeader   string
}

// EmailContentConfiguration holds the configuration for emails, both subjects and template URLs.
type EmailContentConfiguration struct {
	Invite       string `json:"invite" toml:"invite"`
	Confirmation string `json:"confirmation" toml:"confirmation"`
	Recovery     string `json:"recovery" toml:"recovery"`
	EmailChange  string `json:"email_change" toml:"email_change"`
}

type ProviderConfiguration struct {
	Bitbucket   OAuthProviderConfiguration `json:"bitbucket" toml:"bitbucket"`
	Github      OAuthProviderConfiguration `json:"github" toml:"github"`
	Gitlab      OAuthProviderConfiguration `json:"gitlab" toml:"gitlab"`
	Google      OAuthProviderConfiguration `json:"google" toml:"google"`
	Facebook    OAuthProviderConfiguration `json:"facebook" toml:"facebook"`
	Email       EmailProviderConfiguration `json:"email" toml:"email"`
	Saml        SamlProviderConfiguration  `json:"saml" toml:"saml"`
	RedirectURL string                     `json:"redirect_url" toml:"redirect_url"`
}

type SMTPConfiguration struct {
	MaxFrequency time.Duration `json:"max_frequency"`
	Host         string        `json:"host" toml:"host"`
	Port         int           `json:"port,omitempty" toml:"port"`
	User         string        `json:"user" toml:"user"`
	Pass         string        `json:"pass,omitempty" toml:"pass"`
	AdminEmail   string        `json:"admin_email" toml:"admin_email"`
}

type MailerConfiguration struct {
	Autoconfirm        bool                      `json:"autoconfirm"`
	Subjects           EmailContentConfiguration `json:"subjects"`
	Templates          EmailContentConfiguration `json:"templates"`
	URLPaths           EmailContentConfiguration `json:"url_paths"`
	RecoveryMaxAge     time.Duration             `json:"recovery_max_age"`
	ConfirmationMaxAge time.Duration             `json:"confirmation_max_age"`
	InviteMaxAge       time.Duration             `json:"invite_max_age"`
}

// Configuration holds all the per-instance configuration.
type Configuration struct {
	SiteURL       string                `json:"site_url"`
	JWT           JWTConfiguration      `json:"jwt"`
	SMTP          SMTPConfiguration     `json:"smtp"`
	Mailer        MailerConfiguration   `json:"mailer"`
	External      ProviderConfiguration `json:"external"`
	DisableSignup bool                  `json:"disable_signup"`
	Webhook       WebhookConfig         `json:"webhook"`
	Cookie        struct {
		Key      string `json:"key"`
		Duration int    `json:"duration"`
	} `json:"cookies"`
}

type WebhookConfig struct {
	URL        string   `json:"url" toml:"url"`
	Retries    int      `json:"retries" toml:"retries"`
	TimeoutSec int      `json:"timeout_sec" toml:"timeout_sec"`
	Secret     string   `json:"secret" toml:"secret"`
	Events     []string `json:"events" toml:"events"`
}

func (w *WebhookConfig) HasEvent(event string) bool {
	for _, name := range w.Events {
		if event == name {
			return true
		}
	}
	return false
}

// rawSMTP is used for TOML decoding of SMTP config (duration as string).
type rawSMTP struct {
	MaxFrequency string `toml:"max_frequency"`
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	Pass         string `toml:"pass"`
	AdminEmail   string `toml:"admin_email"`
}

// rawMailer is used for TOML decoding of Mailer config (durations as strings).
type rawMailer struct {
	Autoconfirm        bool                      `toml:"autoconfirm"`
	Subjects           EmailContentConfiguration `toml:"subjects"`
	Templates          EmailContentConfiguration `toml:"templates"`
	URLPaths           EmailContentConfiguration `toml:"url_paths"`
	RecoveryMaxAge     string                    `toml:"recovery_max_age"`
	ConfirmationMaxAge string                    `toml:"confirmation_max_age"`
	InviteMaxAge       string                    `toml:"invite_max_age"`
}

// appConfig is the unified TOML configuration structure.
type appConfig struct {
	SiteURL           string `toml:"site_url"`
	OperatorToken     string `toml:"operator_token"`
	DisableSignup     bool   `toml:"disable_signup"`
	RateLimitHeader   string `toml:"rate_limit_header"`
	MultiInstanceMode bool   `toml:"multi_instance_mode"`

	API struct {
		Host            string `toml:"host"`
		Port            int    `toml:"port"`
		Endpoint        string `toml:"endpoint"`
		RequestIDHeader string `toml:"request_id_header"`
		ExportSecret    string `toml:"export_secret"`
	} `toml:"api"`

	DB       DBConfiguration       `toml:"db"`
	JWT      JWTConfiguration      `toml:"jwt"`
	External ProviderConfiguration `toml:"external"`
	Log      LoggingConfig         `toml:"log"`
	Tracing  TracingConfig         `toml:"tracing"`
	Webhook  WebhookConfig         `toml:"webhook"`
	SMTP     rawSMTP               `toml:"smtp"`
	Mailer   rawMailer             `toml:"mailer"`

	Cookie struct {
		Key      string `toml:"key"`
		Duration int    `toml:"duration"`
	} `toml:"cookie"`
}

func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func (raw *appConfig) toSMTPConfiguration() SMTPConfiguration {
	return SMTPConfiguration{
		MaxFrequency: parseDuration(raw.SMTP.MaxFrequency),
		Host:         raw.SMTP.Host,
		Port:         raw.SMTP.Port,
		User:         raw.SMTP.User,
		Pass:         raw.SMTP.Pass,
		AdminEmail:   raw.SMTP.AdminEmail,
	}
}

func loadTOML(filename string) (*appConfig, error) {
	var cfg appConfig
	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadGlobal loads global configuration from a TOML file.
func LoadGlobal(filename string) (*GlobalConfiguration, error) {
	raw, err := loadTOML(filename)
	if err != nil {
		return nil, err
	}

	if raw.DB.MigrationsPath == "" {
		raw.DB.MigrationsPath = "./migrations"
	}

	config := &GlobalConfiguration{
		DB:                raw.DB,
		External:          raw.External,
		Logging:           raw.Log,
		OperatorToken:     raw.OperatorToken,
		MultiInstanceMode: raw.MultiInstanceMode,
		Tracing:           raw.Tracing,
		SMTP:              raw.toSMTPConfiguration(),
		RateLimitHeader:   raw.RateLimitHeader,
	}
	config.API.Host = raw.API.Host
	config.API.Port = raw.API.Port
	if config.API.Port == 0 {
		config.API.Port = 8081
	}
	config.API.Endpoint = raw.API.Endpoint
	config.API.RequestIDHeader = raw.API.RequestIDHeader
	config.API.ExportSecret = raw.API.ExportSecret

	if _, err := ConfigureLogging(&config.Logging); err != nil {
		return nil, err
	}
	ConfigureTracing(&config.Tracing)

	if config.SMTP.MaxFrequency == 0 {
		config.SMTP.MaxFrequency = 15 * time.Minute
	}
	return config, nil
}

// LoadConfig loads per-instance configuration from a TOML file.
func LoadConfig(filename string) (*Configuration, error) {
	raw, err := loadTOML(filename)
	if err != nil {
		return nil, err
	}

	config := &Configuration{
		SiteURL: raw.SiteURL,
		JWT:     raw.JWT,
		SMTP:    raw.toSMTPConfiguration(),
		Mailer: MailerConfiguration{
			Autoconfirm:        raw.Mailer.Autoconfirm,
			Subjects:           raw.Mailer.Subjects,
			Templates:          raw.Mailer.Templates,
			URLPaths:           raw.Mailer.URLPaths,
			RecoveryMaxAge:     parseDuration(raw.Mailer.RecoveryMaxAge),
			ConfirmationMaxAge: parseDuration(raw.Mailer.ConfirmationMaxAge),
			InviteMaxAge:       parseDuration(raw.Mailer.InviteMaxAge),
		},
		External:      raw.External,
		DisableSignup: raw.DisableSignup,
		Webhook:       raw.Webhook,
	}
	config.Cookie.Key = raw.Cookie.Key
	config.Cookie.Duration = raw.Cookie.Duration

	config.ApplyDefaults()
	return config, nil
}

// ApplyDefaults sets defaults for a Configuration
func (config *Configuration) ApplyDefaults() {
	if config.JWT.AdminGroupName == "" {
		config.JWT.AdminGroupName = "admin"
	}

	if config.JWT.Exp == 0 {
		config.JWT.Exp = 3600
	}

	if config.JWT.RefreshTokenLifetime <= 0 {
		config.JWT.RefreshTokenLifetime = 2592000 // 30 days
	}

	if config.Mailer.URLPaths.Invite == "" {
		config.Mailer.URLPaths.Invite = "/"
	}
	if config.Mailer.URLPaths.Confirmation == "" {
		config.Mailer.URLPaths.Confirmation = "/"
	}
	if config.Mailer.URLPaths.Recovery == "" {
		config.Mailer.URLPaths.Recovery = "/"
	}
	if config.Mailer.URLPaths.EmailChange == "" {
		config.Mailer.URLPaths.EmailChange = "/"
	}

	if config.SMTP.MaxFrequency == 0 {
		config.SMTP.MaxFrequency = 15 * time.Minute
	}

	if config.Mailer.RecoveryMaxAge <= 0 {
		config.Mailer.RecoveryMaxAge = 24 * time.Hour
	}

	if config.Mailer.ConfirmationMaxAge <= 0 {
		config.Mailer.ConfirmationMaxAge = 24 * time.Hour
	}

	if config.Mailer.InviteMaxAge <= 0 {
		config.Mailer.InviteMaxAge = 7 * 24 * time.Hour
	}

	if config.Cookie.Key == "" {
		config.Cookie.Key = "nf_jwt"
	}

	if config.Cookie.Duration == 0 {
		config.Cookie.Duration = 86400
	}
}

func (config *Configuration) Value() (driver.Value, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return driver.Value(""), err
	}
	return driver.Value(string(data)), nil
}

func (config *Configuration) Scan(src interface{}) error {
	var source []byte
	switch v := src.(type) {
	case string:
		source = []byte(v)
	case []byte:
		source = v
	default:
		return errors.New("Invalid data type for Configuration")
	}

	if len(source) == 0 {
		source = []byte("{}")
	}
	return json.Unmarshal(source, &config)
}

func (o *OAuthProviderConfiguration) Validate() error {
	if !o.Enabled {
		return errors.New("Provider is not enabled")
	}
	if o.ClientID == "" {
		return errors.New("Missing Oauth client ID")
	}
	if o.Secret == "" {
		return errors.New("Missing Oauth secret")
	}
	if o.RedirectURI == "" {
		return errors.New("Missing redirect URI")
	}
	return nil
}
