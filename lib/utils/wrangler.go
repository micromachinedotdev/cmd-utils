package utils

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/tidwall/jsonc"
)

type WranglerConfig struct {
	Name               string   `toml:"name" json:"name"`
	Main               string   `toml:"main" json:"main"`
	CompatibilityDate  string   `toml:"compatibility_date" json:"compatibility_date"`
	CompatibilityFlags []string `toml:"compatibility_flags" json:"compatibility_flags,omitempty"`

	NoBundle bool `json:""`

	// Worker type
	Type string `toml:"type" json:"type,omitempty"` // "module" or "service-worker" (deprecated)

	// Routes & domains
	Routes []Route `toml:"routes" json:"routes,omitempty"`
	Route  string  `toml:"route" json:"route,omitempty"` // single route shorthand

	// Assets (for static sites / Workers Sites)
	Assets *AssetsConfig `toml:"assets" json:"assets,omitempty"`
	Site   *SiteConfig   `toml:"site" json:"site,omitempty"` // legacy Workers Sites

	// Bindings
	KVNamespaces    []KVNamespace       `toml:"kv_namespaces" json:"kv_namespaces,omitempty"`
	R2Buckets       []R2Bucket          `toml:"r2_buckets" json:"r2_buckets,omitempty"`
	D1Databases     []D1Database        `toml:"d1_databases" json:"d1_databases,omitempty"`
	DurableObjects  *DurableObjects     `toml:"durable_objects" json:"durable_objects,omitempty"`
	Services        []ServiceBinding    `toml:"services" json:"services,omitempty"`
	AnalyticsEngine []AnalyticsBinding  `toml:"analytics_engine_datasets" json:"analytics_engine_datasets,omitempty"`
	Queues          *QueuesConfig       `toml:"queues" json:"queues,omitempty"`
	Hyperdrive      []HyperdriveBinding `toml:"hyperdrive" json:"hyperdrive,omitempty"`
	Vectorize       []VectorizeBinding  `toml:"vectorize" json:"vectorize,omitempty"`
	AIBindings      []AIBinding         `toml:"ai" json:"ai,omitempty"`

	// Variables & secrets
	Vars    map[string]string `toml:"vars" json:"vars,omitempty"`
	Secrets []string          `toml:"-" json:"-"` // not in config, just for reference

	// Build
	Build *BuildConfig `toml:"build" json:"build,omitempty"`

	// Environments
	Env map[string]*WranglerConfig `toml:"env" json:"env,omitempty"`

	// Observability
	Logpush       bool           `toml:"logpush" json:"logpush,omitempty"`
	TailConsumers []TailConsumer `toml:"tail_consumers" json:"tail_consumers,omitempty"`

	// Limits
	Limits *LimitsConfig `toml:"limits" json:"limits,omitempty"`

	// Placement
	Placement *PlacementConfig `toml:"placement" json:"placement,omitempty"`
}

type NormalizedWranglerConfig struct {
	// Meta
	ConfigPath          string   `json:"configPath,omitempty" toml:"-"`
	UserConfigPath      string   `json:"userConfigPath,omitempty" toml:"-"`
	TopLevelName        string   `json:"topLevelName,omitempty" toml:"-"`
	DefinedEnvironments []string `json:"definedEnvironments,omitempty" toml:"-"`

	// Core
	Name               string   `json:"name" toml:"name"`
	Main               string   `json:"main" toml:"main"`
	CompatibilityDate  string   `json:"compatibility_date" toml:"compatibility_date"`
	CompatibilityFlags []string `json:"compatibility_flags,omitempty" toml:"compatibility_flags"`
	LegacyEnv          bool     `json:"legacy_env,omitempty" toml:"legacy_env"`
	NoBundle           bool     `json:"no_bundle,omitempty" toml:"no_bundle"`

	// JSX
	JSXFactory  string `json:"jsx_factory,omitempty" toml:"jsx_factory"`
	JSXFragment string `json:"jsx_fragment,omitempty" toml:"jsx_fragment"`

	// Module rules
	Rules []ModuleRule `json:"rules,omitempty" toml:"rules"`

	// Assets
	Assets *AssetsConfig `json:"assets,omitempty" toml:"assets"`

	// Triggers (crons, etc)
	Triggers TriggersConfig `json:"triggers,omitempty" toml:"triggers"`

	// Variables & secrets
	Vars map[string]string `json:"vars,omitempty" toml:"vars"`

	// Bindings
	KVNamespaces            []KVNamespace        `json:"kv_namespaces,omitempty" toml:"kv_namespaces"`
	R2Buckets               []R2Bucket           `json:"r2_buckets,omitempty" toml:"r2_buckets"`
	D1Databases             []D1Database         `json:"d1_databases,omitempty" toml:"d1_databases"`
	DurableObjects          DurableObjects       `json:"durable_objects,omitempty" toml:"durable_objects"`
	Services                []ServiceBinding     `json:"services,omitempty" toml:"services"`
	AnalyticsEngineDatasets []AnalyticsBinding   `json:"analytics_engine_datasets,omitempty" toml:"analytics_engine_datasets"`
	Queues                  QueuesConfig         `json:"queues,omitempty" toml:"queues"`
	Hyperdrive              []HyperdriveBinding  `json:"hyperdrive,omitempty" toml:"hyperdrive"`
	Vectorize               []VectorizeBinding   `json:"vectorize,omitempty" toml:"vectorize"`
	Workflows               []WorkflowBinding    `json:"workflows,omitempty" toml:"workflows"`
	Migrations              []Migration          `json:"migrations,omitempty" toml:"migrations"`
	MTLSCertificates        []MTLSCertificate    `json:"mtls_certificates,omitempty" toml:"mtls_certificates"`
	SendEmail               []SendEmailBinding   `json:"send_email,omitempty" toml:"send_email"`
	DispatchNamespaces      []DispatchNamespace  `json:"dispatch_namespaces,omitempty" toml:"dispatch_namespaces"`
	Pipelines               []PipelineBinding    `json:"pipelines,omitempty" toml:"pipelines"`
	SecretsStoreSecrets     []SecretStoreBinding `json:"secrets_store_secrets,omitempty" toml:"secrets_store_secrets"`
	Ratelimits              []RatelimitBinding   `json:"ratelimits,omitempty" toml:"ratelimits"`
	VPCServices             []VPCService         `json:"vpc_services,omitempty" toml:"vpc_services"`
	WorkerLoaders           []WorkerLoader       `json:"worker_loaders,omitempty" toml:"worker_loaders"`

	// Legacy/misc
	Cloudchamber     json.RawMessage `json:"cloudchamber,omitempty" toml:"cloudchamber"`
	Logfwdr          LogfwdrConfig   `json:"logfwdr,omitempty" toml:"logfwdr"`
	UnsafeHelloWorld []any           `json:"unsafe_hello_world,omitempty" toml:"-"`

	// Observability
	Observability ObservabilityConfig `json:"observability,omitempty" toml:"observability"`

	// Python
	PythonModules PythonModulesConfig `json:"python_modules,omitempty" toml:"python_modules"`

	// Dev settings
	Dev DevConfig `json:"dev,omitempty" toml:"dev"`

	// Environments
	Env map[string]*WranglerConfig `json:"env,omitempty" toml:"env"`
}

type ModuleRule struct {
	Type  string   `json:"type" toml:"type"` // "ESModule", "CommonJS", "Text", etc.
	Globs []string `json:"globs" toml:"globs"`
}

type AssetsConfig struct {
	Directory        string `json:"directory,omitempty" toml:"directory"`
	Binding          string `json:"binding,omitempty" toml:"binding"`
	HTMLHandling     string `json:"html_handling,omitempty" toml:"html_handling"`
	NotFoundHandling string `json:"not_found_handling,omitempty" toml:"not_found_handling"`
}

type TriggersConfig struct {
	Crons []string `json:"crons,omitempty" toml:"crons"`
}

type DurableObjects struct {
	Bindings []DurableObjectBinding `json:"bindings,omitempty" toml:"bindings"`
}

type DurableObjectBinding struct {
	Name       string `json:"name" toml:"name"`
	ClassName  string `json:"class_name" toml:"class_name"`
	ScriptName string `json:"script_name,omitempty" toml:"script_name"`
}

type QueuesConfig struct {
	Producers []QueueProducer `json:"producers,omitempty" toml:"producers"`
	Consumers []QueueConsumer `json:"consumers,omitempty" toml:"consumers"`
}

type QueueProducer struct {
	Binding string `json:"binding" toml:"binding"`
	Queue   string `json:"queue" toml:"queue"`
}

type QueueConsumer struct {
	Queue           string `json:"queue" toml:"queue"`
	MaxBatchSize    int    `json:"max_batch_size,omitempty" toml:"max_batch_size"`
	MaxBatchTimeout int    `json:"max_batch_timeout,omitempty" toml:"max_batch_timeout"`
	MaxRetries      int    `json:"max_retries,omitempty" toml:"max_retries"`
	DeadLetterQueue string `json:"dead_letter_queue,omitempty" toml:"dead_letter_queue"`
}

type KVNamespace struct {
	Binding   string `json:"binding" toml:"binding"`
	ID        string `json:"id,omitempty" toml:"id"`
	PreviewID string `json:"preview_id,omitempty" toml:"preview_id"`
}

type R2Bucket struct {
	Binding           string `json:"binding" toml:"binding"`
	BucketName        string `json:"bucket_name" toml:"bucket_name"`
	PreviewBucketName string `json:"preview_bucket_name,omitempty" toml:"preview_bucket_name"`
}

type D1Database struct {
	Binding      string `json:"binding" toml:"binding"`
	DatabaseName string `json:"database_name" toml:"database_name"`
	DatabaseID   string `json:"database_id" toml:"database_id"`
}

type ServiceBinding struct {
	Binding     string `json:"binding" toml:"binding"`
	Service     string `json:"service" toml:"service"`
	Environment string `json:"environment,omitempty" toml:"environment"`
}

type HyperdriveBinding struct {
	Binding string `json:"binding" toml:"binding"`
	ID      string `json:"id" toml:"id"`
}

type VectorizeBinding struct {
	Binding   string `json:"binding" toml:"binding"`
	IndexName string `json:"index_name" toml:"index_name"`
}

type AnalyticsBinding struct {
	Binding string `json:"binding" toml:"binding"`
	Dataset string `json:"dataset,omitempty" toml:"dataset"`
}

type WorkflowBinding struct {
	Binding   string `json:"binding" toml:"binding"`
	Name      string `json:"name" toml:"name"`
	ClassName string `json:"class_name" toml:"class_name"`
}

type Migration struct {
	Tag            string         `json:"tag" toml:"tag"`
	NewClasses     []string       `json:"new_classes,omitempty" toml:"new_classes"`
	DeletedClasses []string       `json:"deleted_classes,omitempty" toml:"deleted_classes"`
	RenamedClasses []RenamedClass `json:"renamed_classes,omitempty" toml:"renamed_classes"`
}

type RenamedClass struct {
	From string `json:"from" toml:"from"`
	To   string `json:"to" toml:"to"`
}

type MTLSCertificate struct {
	Binding       string `json:"binding" toml:"binding"`
	CertificateID string `json:"certificate_id" toml:"certificate_id"`
}

type SendEmailBinding struct {
	Binding             string   `json:"binding" toml:"binding"`
	DestinationAddress  string   `json:"destination_address,omitempty" toml:"destination_address"`
	AllowedDestinations []string `json:"allowed_destinations,omitempty" toml:"allowed_destinations"`
}

type DispatchNamespace struct {
	Binding   string `json:"binding" toml:"binding"`
	Namespace string `json:"namespace" toml:"namespace"`
}

type PipelineBinding struct {
	Binding  string `json:"binding" toml:"binding"`
	Pipeline string `json:"pipeline" toml:"pipeline"`
}

type SecretStoreBinding struct {
	Binding string `json:"binding" toml:"binding"`
	StoreID string `json:"store_id" toml:"store_id"`
}

type RatelimitBinding struct {
	Binding   string `json:"binding" toml:"binding"`
	Namespace string `json:"namespace" toml:"namespace"`
}

type VPCService struct {
	Binding string `json:"binding" toml:"binding"`
	Service string `json:"service" toml:"service"`
}

type WorkerLoader struct {
	Binding string `json:"binding" toml:"binding"`
	Script  string `json:"script" toml:"script"`
}

type LogfwdrConfig struct {
	Bindings []LogfwdrBinding `json:"bindings,omitempty" toml:"bindings"`
}

type LogfwdrBinding struct {
	Name        string `json:"name" toml:"name"`
	Destination string `json:"destination" toml:"destination"`
}

type ObservabilityConfig struct {
	Enabled bool `json:"enabled,omitempty" toml:"enabled"`
}

type PythonModulesConfig struct {
	Exclude []string `json:"exclude,omitempty" toml:"exclude"`
}

type DevConfig struct {
	IP               string `json:"ip,omitempty" toml:"ip"`
	Port             int    `json:"port,omitempty" toml:"port"`
	LocalProtocol    string `json:"local_protocol,omitempty" toml:"local_protocol"`
	UpstreamProtocol string `json:"upstream_protocol,omitempty" toml:"upstream_protocol"`
	EnableContainers bool   `json:"enable_containers,omitempty" toml:"enable_containers"`
	GenerateTypes    bool   `json:"generate_types,omitempty" toml:"generate_types"`
}

type Route struct {
	Pattern      string `toml:"pattern" json:"pattern"`
	ZoneID       string `toml:"zone_id" json:"zone_id,omitempty"`
	ZoneName     string `toml:"zone_name" json:"zone_name,omitempty"`
	CustomDomain bool   `toml:"custom_domain" json:"custom_domain,omitempty"`
}

type SiteConfig struct {
	Bucket     string   `toml:"bucket" json:"bucket"`
	EntryPoint string   `toml:"entry-point" json:"entry-point,omitempty"`
	Include    []string `toml:"include" json:"include,omitempty"`
	Exclude    []string `toml:"exclude" json:"exclude,omitempty"`
}

type AIBinding struct {
	Binding string `toml:"binding" json:"binding"`
}

type BuildConfig struct {
	Command  string `toml:"command" json:"command,omitempty"`
	Cwd      string `toml:"cwd" json:"cwd,omitempty"`
	WatchDir string `toml:"watch_dir" json:"watch_dir,omitempty"`
}

type TailConsumer struct {
	Service     string `toml:"service" json:"service"`
	Environment string `toml:"environment" json:"environment,omitempty"`
}

type LimitsConfig struct {
	CPUMs int `toml:"cpu_ms" json:"cpu_ms,omitempty"`
}

type PlacementConfig struct {
	Mode string `toml:"mode" json:"mode,omitempty"` // "smart"
}

func DetectWranglerFile[T any](root *string) (*T, error) {
	rootDir := ""
	if root != nil {
		rootDir = *root
	}

	paths := []string{
		filepath.Join(rootDir, "/wrangler.toml"),
		filepath.Join(rootDir, "/wrangler.json"),
		filepath.Join(rootDir, "/wrangler.jsonc"),
	}

	content := ""
	usedPath := ""

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		usedPath = path
		content = string(data)
	}

	if content == "" || usedPath == "" {
		return nil, errors.New("no wrangler configuration file found")
	}

	ext := filepath.Ext(usedPath)
	if ext == "" {
		return nil, errors.New("invalid wrangler configuration file")
	}

	config := new(T)
	switch ext {
	case ".json", ".jsonc":
		err := json.Unmarshal(jsonc.ToJSON([]byte(content)), &config)
		if err != nil {
			return nil, err
		}
	case ".toml":
		err := toml.Unmarshal([]byte(content), &config)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid wrangler configuration file")
	}

	return config, nil
}
