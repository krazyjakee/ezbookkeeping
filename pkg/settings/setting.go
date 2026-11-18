package settings

import (
	"net"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/ini.v1"

	"github.com/mayswind/ezbookkeeping/pkg/core"
)

const (
	ebkWorkDirEnvName                  = "EBK_WORK_DIR"
	ebkConfigItemValueEnvNamePrefix    = "EBK"
	ebkConfigItemFilePathEnvNamePrefix = "EBKCFP"
	defaultConfigPath                  = "/conf/ezbookkeeping.ini"
	defaultRootUrl                     = "%(protocol)s://%(domain)s:%(http_port)s/"
	defaultStaticRootPath              = "public"
)

// SystemMode represents running mode of system
type SystemMode string

// System running modes
const (
	MODE_DEVELOPMENT SystemMode = "development"
	MODE_PRODUCTION  SystemMode = "production"
)

// Scheme represents how the web backend service exposes
type Scheme string

// Scheme types
const (
	SCHEME_HTTP   Scheme = "http"
	SCHEME_HTTPS  Scheme = "https"
	SCHEME_SOCKET Scheme = "socket"
)

// Level represents log level
type Level string

// Log levels
const (
	LOGLEVEL_DEBUG Level = "debug"
	LOGLEVEL_INFO  Level = "info"
	LOGLEVEL_WARN  Level = "warn"
	LOGLEVEL_ERROR Level = "error"
)

// Database types
const (
	MySqlDbType    string = "mysql"
	PostgresDbType string = "postgres"
	Sqlite3DbType  string = "sqlite3"
)

// Object Storage types
const (
	LocalFileSystemObjectStorageType string = "local_filesystem"
	S3StorageType                    string = "s3"
	MinIOStorageType                 string = "minio"
	WebDAVStorageType                string = "webdav"
)

const (
	OpenAILLMProvider                    string = "openai"
	OpenAICompatibleLLMProvider          string = "openai_compatible"
	OpenAIResponsesCompatibleLLMProvider string = "openai_responses_compatible"
	AnthropicLLMProvider                 string = "anthropic"
	AnthropicCompatibleLLMProvider       string = "anthropic_compatible"
	OpenRouterLLMProvider                string = "openrouter"
	OllamaLLMProvider                    string = "ollama"
	LMStudioLLMProvider                  string = "lm_studio"
	GoogleAILLMProvider                  string = "google_ai"
	DonutLLMProvider                     string = "donut"
)

// LLMThinkingLevel represents the thinking level of a large language model
type LLMThinkingLevel string

// LLM thinking levels
const (
	LLMThinkingDefault  LLMThinkingLevel = ""
	LLMThinkingDisabled LLMThinkingLevel = "off"
	LLMThinkingEnabled  LLMThinkingLevel = "on"
	LLMThinkingLow      LLMThinkingLevel = "low"
	LLMThinkingMedium   LLMThinkingLevel = "medium"
	LLMThinkingHigh     LLMThinkingLevel = "high"
	LLMThinkingXHigh    LLMThinkingLevel = "xhigh"
)

// Uuid generator types
const (
	InternalUuidGeneratorType string = "internal"
)

// Duplicate checker types
const (
	InMemoryDuplicateCheckerType string = "in_memory"
)

// OAuth 2.0 user identifier types
const (
	OAuth2UserIdentifierEmail    string = "email"
	OAuth2UserIdentifierUsername string = "username"
)

// OAuth 2.0 provider types
const (
	OAuth2ProviderOIDC      string = "oidc"
	OAuth2ProviderNextcloud string = "nextcloud"
	OAuth2ProviderGithub    string = "github"
)

// Map provider types
const (
	OpenStreetMapProvider                  string = "openstreetmap"
	OpenStreetMapHumanitarianStyleProvider string = "openstreetmap_humanitarian"
	OpenTopoMapProvider                    string = "opentopomap"
	OPNVKarteMapProvider                   string = "opnvkarte"
	CyclOSMMapProvider                     string = "cyclosm"
	CartoDBMapProvider                     string = "cartodb"
	TomTomMapProvider                      string = "tomtom"
	TianDiTuProvider                       string = "tianditu"
	GoogleMapProvider                      string = "googlemap"
	BaiduMapProvider                       string = "baidumap"
	AmapProvider                           string = "amap"
	CustomProvider                         string = "custom"
)

// Amap security verification method
const (
	AmapSecurityVerificationInternalProxyMethod string = "internal_proxy"
	AmapSecurityVerificationExternalProxyMethod string = "external_proxy"
	AmapSecurityVerificationPlainTextMethod     string = "plain_text"
)

// Exchange rates data source types
const (
	CentralBankOfArgentinaDataSource   string = "central_bank_of_argentina"
	NationalBankOfBelarusDataSource    string = "national_bank_of_belarus"
	BankOfCanadaDataSource             string = "bank_of_canada"
	CzechNationalBankDataSource        string = "czech_national_bank"
	DanmarksNationalbankDataSource     string = "danmarks_national_bank"
	EuroCentralBankDataSource          string = "euro_central_bank"
	NationalBankOfGeorgiaDataSource    string = "national_bank_of_georgia"
	CentralBankOfHungaryDataSource     string = "central_bank_of_hungary"
	BankOfIsraelDataSource             string = "bank_of_israel"
	BankOfItalyDataSource              string = "bank_of_italy"
	NationalBankOfKazakhstanDataSource string = "national_bank_of_kazakhstan"
	CentralBankOfMalaysiaDataSource    string = "central_bank_of_malaysia"
	CentralBankOfMyanmarDataSource     string = "central_bank_of_myanmar"
	NorgesBankDataSource               string = "norges_bank"
	NationalBankOfPolandDataSource     string = "national_bank_of_poland"
	NationalBankOfRomaniaDataSource    string = "national_bank_of_romania"
	BankOfRussiaDataSource             string = "bank_of_russia"
	SwissNationalBankDataSource        string = "swiss_national_bank"
	NationalBankOfUkraineDataSource    string = "national_bank_of_ukraine"
	CentralBankOfUzbekistanDataSource  string = "central_bank_of_uzbekistan"
	UserCustomExchangeRatesDataSource  string = "user_custom"
)

const (
	defaultHttpAddr string = "0.0.0.0"
	defaultHttpPort uint16 = 8080
	defaultDomain   string = "localhost"

	defaultDatabaseHost            string = "127.0.0.1:3306"
	defaultDatabaseName            string = "ezbookkeeping"
	defaultDatabaseMaxIdleConn     uint16 = 2
	defaultDatabaseMaxOpenConn     uint16 = 0
	defaultDatabaseConnMaxLifetime uint32 = 14400

	defaultLogMode        string = "console"
	defaultLogFileMaxSize uint32 = 104857600 // 100 MB
	defaultLogFileMaxDays uint32 = 7         // days

	defaultWebDAVRequestTimeout uint32 = 10000 // 10 seconds

	defaultAIRecognitionPictureMaxSize                 uint32 = 10485760 // 10MB
	defaultAnthropicLargeLanguageModelAPIMaximumTokens uint32 = 1024
	defaultAnthropicLargeLanguageModelThinkingBudget   uint32 = 1024
	defaultLargeLanguageModelAPIRequestTimeout         uint32 = 60000 // 60 seconds

	defaultDonutModelDownloadURL       string = "https://huggingface.co/krazyjakee/ezbookkeeping-donut-onnx/resolve/874b064ee4bfada52b45250e3f5507fc456d0a32/"
	defaultDonutModelPath              string = "data/donut"
	defaultDonutOnnxRuntimeLibraryPath string = "libonnxruntime.so.1"
	defaultDonutThreads                uint32 = 4
	defaultDonutWorkerIdleTimeout      uint32 = 600 // 10 minutes

	defaultInMemoryDuplicateCheckerCleanupInterval uint32 = 60  // 1 minutes
	defaultDuplicateSubmissionsInterval            uint32 = 300 // 5 minutes

	defaultSecretKey                          string = "ezbookkeeping"
	defaultTrustedProxyIPs                    string = "10.0.0.0/8,169.254.0.0/16,127.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
	defaultTokenExpiredTime                   uint32 = 2592000 // 30 days
	defaultTokenMinRefreshInterval            uint32 = 86400   // 1 day
	defaultTemporaryTokenExpiredTime          uint32 = 300     // 5 minutes
	defaultEmailVerifyTokenExpiredTime        uint32 = 3600    // 60 minutes
	defaultUserAvatarTokenExpiredTime         uint32 = 3600    // 60 minutes
	defaultUserCustomIconTokenExpiredTime     uint32 = 3600    // 60 minutes
	defaultTransactionPictureTokenExpiredTime uint32 = 3600    // 60 minutes
	defaultPasswordResetTokenExpiredTime      uint32 = 3600    // 60 minutes
	defaultMaxFailuresPerIpPerMinute          uint32 = 5
	defaultMaxFailuresPerUserPerMinute        uint32 = 5

	defaultOAuth2StateExpiredTime uint32 = 300   // 5 minutes
	defaultOAuth2RequestTimeout   uint32 = 10000 // 10 seconds

	defaultUserCustomIconFileMaxSize     uint32 = 1048576  // 1MB
	defaultTransactionPictureFileMaxSize uint32 = 10485760 // 10MB
	defaultUserAvatarFileMaxSize         uint32 = 1048576  // 1MB

	defaultImportFileMaxSize uint32 = 10485760 // 10MB

	defaultBackupCron           string = "0 3 * * *" // every day at 03:00
	defaultBackupPath           string = "data/backups"
	defaultBackupRetentionCount uint32 = 7

	defaultExchangeRatesDataRequestTimeout uint32 = 10000 // 10 seconds
)

// DatabaseConfig represents the database setting config
type DatabaseConfig struct {
	DatabaseType     string
	DatabaseHost     string
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string

	DatabaseSSLMode string

	DatabasePath string

	MaxIdleConnection     uint16
	MaxOpenConnection     uint16
	ConnectionMaxLifeTime uint32
}

// SMTPConfig represents the SMTP setting config
type SMTPConfig struct {
	SMTPHost          string
	SMTPUser          string
	SMTPPasswd        string
	SMTPSkipTLSVerify bool
	FromAddress       string
}

// S3Config represents the S3-compatible object storage setting config
type S3Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	UseSSL          bool
	SkipTLSVerify   bool
	UsePathStyle    bool
	Bucket          string
	RootPath        string
}

// MinIOConfig represents the MinIO setting config
type MinIOConfig struct {
	Endpoint        string
	Location        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	SkipTLSVerify   bool
	Bucket          string
	RootPath        string
}

// WebDAVConfig represents the WebDAV setting config
type WebDAVConfig struct {
	Url            string
	Username       string
	Password       string
	RootPath       string
	RequestTimeout uint32
	Proxy          string
	SkipTLSVerify  bool
}

// LLMConfig represents the Large Language Model setting config
type LLMConfig struct {
	LLMProvider                             string
	EnableThinking                          LLMThinkingLevel
	OpenAIAPIKey                            string
	OpenAIModelID                           string
	OpenAICompatibleBaseURL                 string
	OpenAICompatibleAPIKey                  string
	OpenAICompatibleModelID                 string
	AnthropicAPIKey                         string
	AnthropicModelID                        string
	AnthropicMaxTokens                      uint32
	AnthropicThinkingBudgetTokens           uint32
	AnthropicCompatibleBaseURL              string
	AnthropicCompatibleAPIVersion           string
	AnthropicCompatibleAPIKey               string
	AnthropicCompatibleModelID              string
	AnthropicCompatibleMaxTokens            uint32
	AnthropicCompatibleThinkingBudgetTokens uint32
	OpenRouterAPIKey                        string
	OpenRouterModelID                       string
	OllamaServerURL                         string
	OllamaModelID                           string
	LMStudioServerURL                       string
	LMStudioToken                           string
	LMStudioModelID                         string
	GoogleAIAPIKey                          string
	GoogleAIModelID                         string
	DonutModelDownloadURL                   string
	DonutModelPath                          string
	DonutWorkerPath                         string
	DonutOnnxRuntimeLibraryPath             string
	DonutThreads                            uint32
	DonutLowMemory                          bool
	DonutWorkerIdleTimeout                  uint32
	DonutDateOrder                          string
	DonutDefaultAccount                     string
	LargeLanguageModelAPIRequestTimeout     uint32
	LargeLanguageModelAPIProxy              string
	LargeLanguageModelAPISkipTLSVerify      bool
}

// MultiLanguageContentConfig represents a multi-language content setting config
type MultiLanguageContentConfig struct {
	Enabled              bool
	DefaultContent       string
	MultiLanguageContent map[string]string
}

// Config represents the global setting config
type Config struct {
	// Global
	Mode           SystemMode
	WorkingPath    string
	ConfigFilePath string

	// Server
	Protocol Scheme
	HttpAddr string
	HttpPort uint16

	Domain  string
	RootUrl string

	CertFile    string
	CertKeyFile string

	UnixSocketPath string

	StaticRootPath string

	EnableGZip            bool
	EnableRequestLog      bool
	EnableRequestIdHeader bool

	// MCP
	EnableMCPServer     bool
	MCPAllowedRemoteIPs []*core.IPPattern

	// Database
	DatabaseConfig     *DatabaseConfig
	EnableQueryLog     bool
	AutoUpdateDatabase bool

	// Mail
	EnableSMTP bool
	SMTPConfig *SMTPConfig

	// Log
	LogModes         []string
	EnableConsoleLog bool
	EnableFileLog    bool

	EnableDebugLog     bool
	LogLevel           Level
	FileLogPath        string
	RequestFileLogPath string
	QueryFileLogPath   string
	LogFileRotate      bool
	LogFileMaxSize     uint32
	LogFileMaxDays     uint32

	// Storage
	StorageType         string
	LocalFileSystemPath string
	S3Config            *S3Config
	MinIOConfig         *MinIOConfig
	WebDAVConfig        *WebDAVConfig

	// Large Language Model
	TransactionFromAITextRecognition  bool
	TransactionFromAIImageRecognition bool
	InsightsExplorerCodingAssistant   bool
	MaxAIRecognitionPictureFileSize   uint32

	// Large Language Model for Transaction Text Recognition
	TextRecognitionLLMConfig *LLMConfig

	// Large Language Model for Receipt Image Recognition
	ReceiptImageRecognitionLLMConfig *LLMConfig

	// Large Language Model for coding assistant
	CodingAssistantLLMConfig *LLMConfig

	// Uuid
	UuidGeneratorType string
	UuidServerId      uint8

	// Duplicate Checker
	DuplicateCheckerType                            string
	InMemoryDuplicateCheckerCleanupInterval         uint32
	InMemoryDuplicateCheckerCleanupIntervalDuration time.Duration
	EnableDuplicateSubmissionsCheck                 bool
	DuplicateSubmissionsInterval                    uint32
	DuplicateSubmissionsIntervalDuration            time.Duration

	// Cron
	EnableRemoveExpiredTokens        bool
	EnableCreateScheduledTransaction bool

	// Backup
	EnableScheduledBackup bool
	BackupCron            string
	BackupPath            string
	BackupIncludeConfig   bool
	BackupEncrypt         bool
	BackupPassphrase      string
	BackupRetentionCount  uint32
	BackupRetentionDays   uint32

	// Secret
	SecretKeyNoSet                             bool
	SecretKey                                  string
	TrustedProxyIPs                            []*net.IPNet
	TrustedProxyTextualIPs                     []string
	TokenExpiredTime                           uint32
	TokenExpiredTimeDuration                   time.Duration
	TokenMinRefreshInterval                    uint32
	TemporaryTokenExpiredTime                  uint32
	TemporaryTokenExpiredTimeDuration          time.Duration
	EmailVerifyTokenExpiredTime                uint32
	EmailVerifyTokenExpiredTimeDuration        time.Duration
	PasswordResetTokenExpiredTime              uint32
	PasswordResetTokenExpiredTimeDuration      time.Duration
	UserAvatarTokenExpiredTime                 uint32
	UserAvatarTokenExpiredTimeDuration         time.Duration
	UserCustomIconTokenExpiredTime             uint32
	UserCustomIconTokenExpiredTimeDuration     time.Duration
	TransactionPictureTokenExpiredTime         uint32
	TransactionPictureTokenExpiredTimeDuration time.Duration
	EnableAPIToken                             bool
	APITokenAllowedRemoteIPs                   []*core.IPPattern
	MaxFailuresPerIpPerMinute                  uint32
	MaxFailuresPerUserPerMinute                uint32

	// Auth
	EnableInternalAuth                bool
	EnableOAuth2Login                 bool
	EnableTwoFactor                   bool
	EnableUserForgetPassword          bool
	ForgetPasswordRequireVerifyEmail  bool
	OAuth2ClientID                    string
	OAuth2ClientSecret                string
	OAuth2UsePKCE                     bool
	OAuth2UserIdentifier              string
	OAuth2AutoRegister                bool
	OAuth2Provider                    string
	OAuth2StateExpiredTime            uint32
	OAuth2StateExpiredTimeDuration    time.Duration
	OAuth2RequestTimeout              uint32
	OAuth2Proxy                       string
	OAuth2SkipTLSVerify               bool
	OAuth2OIDCProviderIssuerURL       string
	OAuth2OIDCProviderCheckIssuerURL  bool
	OAuth2OIDCCustomDisplayNameConfig MultiLanguageContentConfig
	OAuth2NextcloudBaseUrl            string

	// User
	EnableUserRegister            bool
	RegisterAllowedEmailDomains   []string
	EnableUserVerifyEmail         bool
	EnableUserForceVerifyEmail    bool
	EnableUserCustomIcon          bool
	MaxUserCustomIconFileSize     uint32
	EnableTransactionPictures     bool
	MaxTransactionPictureFileSize uint32
	EnableScheduledTransaction    bool
	AvatarProvider                core.UserAvatarProviderType
	MaxAvatarFileSize             uint32
	DefaultFeatureRestrictions    core.UserFeatureRestrictions

	// Data
	EnableDataExport  bool
	EnableDataImport  bool
	MaxImportFileSize uint32

	// Tip
	LoginPageTips MultiLanguageContentConfig

	// Notification
	AfterRegisterNotification MultiLanguageContentConfig
	AfterLoginNotification    MultiLanguageContentConfig
	AfterOpenNotification     MultiLanguageContentConfig

	// Map
	MapProvider                           string
	EnableMapDataFetchProxy               bool
	SupportMapDataFetchProxy              bool
	MapProxy                              string
	TomTomMapAPIKey                       string
	TianDiTuAPIKey                        string
	GoogleMapAPIKey                       string
	BaiduMapAK                            string
	AmapApplicationKey                    string
	AmapSecurityVerificationMethod        string
	AmapApplicationSecret                 string
	AmapApiExternalProxyUrl               string
	CustomMapTileServerTileLayerUrl       string
	CustomMapTileServerAnnotationLayerUrl string
	CustomMapTileServerMinZoomLevel       uint8
	CustomMapTileServerMaxZoomLevel       uint8
	CustomMapTileServerDefaultZoomLevel   uint8

	// Exchange Rates
	ExchangeRatesDataSource                       string
	ExchangeRatesRequestTimeout                   uint32
	ExchangeRatesRequestTimeoutExceedDefaultValue bool
	ExchangeRatesProxy                            string
	ExchangeRatesSkipTLSVerify                    bool

	// Developer Tools
	EnableLanguagePreview bool
}

// LoadConfiguration loads setting config from given config file path
func LoadConfiguration(configFilePath string) (*Config, error) {
	var err error

	cfgFile, err := ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
	}, configFilePath)

	if err != nil {
		return nil, err
	}

	config := &Config{}
	config.WorkingPath, err = getWorkingPath()

	if err != nil {
		return nil, err
	}

	if absoluteConfigFilePath, err := filepath.Abs(configFilePath); err == nil {
		config.ConfigFilePath = absoluteConfigFilePath
	} else {
		config.ConfigFilePath = configFilePath
	}

	loaders := []struct {
		sectionName string
		load        func(config *Config, configFile *ini.File, sectionName string) error
	}{
		{"global", loadGlobalConfiguration},
		{"server", loadServerConfiguration},
		{"mcp", loadMCPServerConfiguration},
		{"database", loadDatabaseConfiguration},
		{"mail", loadMailConfiguration},
		{"log", loadLogConfiguration},
		{"storage", loadStorageConfiguration},
		{"ai", loadAIConfiguration},
		{"llm_text_recognition", func(config *Config, configFile *ini.File, sectionName string) (err error) {
			config.TextRecognitionLLMConfig, err = loadLLMConfiguration(configFile, sectionName, config.WorkingPath, "")
			return err
		}},
		{"llm_image_recognition", func(config *Config, configFile *ini.File, sectionName string) (err error) {
			config.ReceiptImageRecognitionLLMConfig, err = loadLLMConfiguration(configFile, sectionName, config.WorkingPath, DonutLLMProvider)
			return err
		}},
		{"llm_coding_assistant", func(config *Config, configFile *ini.File, sectionName string) (err error) {
			config.CodingAssistantLLMConfig, err = loadLLMConfiguration(configFile, sectionName, config.WorkingPath, "")
			return err
		}},
		{"uuid", loadUuidConfiguration},
		{"duplicate_checker", loadDuplicateCheckerConfiguration},
		{"cron", loadCronConfiguration},
		{"backup", loadBackupConfiguration},
		{"security", loadSecurityConfiguration},
		{"auth", loadAuthConfiguration},
		{"user", loadUserConfiguration},
		{"data", loadDataConfiguration},
		{"tip", loadTipConfiguration},
		{"notification", loadNotificationConfiguration},
		{"map", loadMapConfiguration},
		{"exchange_rates", loadExchangeRatesConfiguration},
		{"developer", loadDeveloperTools},
	}

	for _, loader := range loaders {
		if err = loader.load(config, cfgFile, loader.sectionName); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// GetDefaultConfigFilePath returns the defaule config file path
func GetDefaultConfigFilePath() (string, error) {
	workingPath, err := getWorkingPath()

	if err != nil {
		return "", err
	}

	cfgFilePath := filepath.Join(workingPath, defaultConfigPath)
	_, err = os.Stat(cfgFilePath)

	if err != nil {
		return "", err
	}

	return cfgFilePath, nil
}
