package settings

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gopkg.in/ini.v1"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
)

func loadGlobalConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	if getConfigItemStringValue(configFile, sectionName, "mode") == "production" {
		config.Mode = MODE_PRODUCTION
	} else if getConfigItemStringValue(configFile, sectionName, "mode") == "development" {
		config.Mode = MODE_DEVELOPMENT
	} else {
		return errs.ErrInvalidServerMode
	}

	return nil
}

func loadServerConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	if getConfigItemStringValue(configFile, sectionName, "protocol") == "http" {
		config.Protocol = SCHEME_HTTP

		config.HttpAddr = getConfigItemStringValue(configFile, sectionName, "http_addr", defaultHttpAddr)
		config.HttpPort = getConfigItemUint16Value(configFile, sectionName, "http_port", defaultHttpPort)
	} else if getConfigItemStringValue(configFile, sectionName, "protocol") == "https" {
		config.Protocol = SCHEME_HTTPS

		config.HttpAddr = getConfigItemStringValue(configFile, sectionName, "http_addr", defaultHttpAddr)
		config.HttpPort = getConfigItemUint16Value(configFile, sectionName, "http_port", defaultHttpPort)

		config.CertFile = getConfigItemStringValue(configFile, sectionName, "cert_file")
		config.CertKeyFile = getConfigItemStringValue(configFile, sectionName, "cert_key_file")
	} else if getConfigItemStringValue(configFile, sectionName, "protocol") == "socket" {
		config.Protocol = SCHEME_SOCKET

		config.UnixSocketPath = getConfigItemStringValue(configFile, sectionName, "unix_socket")
	} else {
		return errs.ErrInvalidProtocol
	}

	config.Domain = getConfigItemStringValue(configFile, sectionName, "domain", defaultDomain)
	config.RootUrl = getConfigItemStringValue(configFile, sectionName, "root_url", defaultRootUrl)
	config.RootUrl = strings.ReplaceAll(config.RootUrl, "%(protocol)s", string(config.Protocol))
	config.RootUrl = strings.ReplaceAll(config.RootUrl, "%(domain)s", config.Domain)
	config.RootUrl = strings.ReplaceAll(config.RootUrl, "%(http_port)s", strconv.Itoa(int(config.HttpPort)))

	if config.RootUrl[len(config.RootUrl)-1] != '/' {
		config.RootUrl += "/"
	}

	staticRootPath := getConfigItemStringValue(configFile, sectionName, "static_root_path", defaultStaticRootPath)
	finalStaticRootPath, err := getFinalPath(config.WorkingPath, staticRootPath)

	if err != nil {
		return err
	}

	config.StaticRootPath = finalStaticRootPath

	config.EnableGZip = getConfigItemBoolValue(configFile, sectionName, "enable_gzip", false)
	config.EnableRequestLog = getConfigItemBoolValue(configFile, sectionName, "log_request", false)
	config.EnableRequestIdHeader = getConfigItemBoolValue(configFile, sectionName, "request_id_header", true)

	return nil
}

func loadMCPServerConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	var err error

	config.EnableMCPServer = getConfigItemBoolValue(configFile, sectionName, "enable_mcp", false)
	config.MCPAllowedRemoteIPs, err = getIPPatterns(configFile, sectionName, "mcp_allowed_remote_ips", "")

	if err != nil {
		return err
	}

	return nil
}

func loadDatabaseConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	dbConfig := &DatabaseConfig{}

	dbConfig.DatabaseType = getConfigItemStringValue(configFile, sectionName, "type")

	if dbConfig.DatabaseType != MySqlDbType &&
		dbConfig.DatabaseType != PostgresDbType &&
		dbConfig.DatabaseType != Sqlite3DbType {
		return errs.ErrDatabaseTypeInvalid
	}

	dbConfig.DatabaseHost = getConfigItemStringValue(configFile, sectionName, "host", defaultDatabaseHost)
	dbConfig.DatabaseName = getConfigItemStringValue(configFile, sectionName, "name", defaultDatabaseName)
	dbConfig.DatabaseUser = getConfigItemStringValue(configFile, sectionName, "user")
	dbConfig.DatabasePassword = getConfigItemStringValue(configFile, sectionName, "passwd")

	if dbConfig.DatabaseType == PostgresDbType {
		dbConfig.DatabaseSSLMode = getConfigItemStringValue(configFile, sectionName, "ssl_mode")
	}

	if dbConfig.DatabaseType == Sqlite3DbType {
		staticDBPath := getConfigItemStringValue(configFile, sectionName, "db_path")
		finalStaticDBPath, _ := getFinalPath(config.WorkingPath, staticDBPath)
		dbConfig.DatabasePath = finalStaticDBPath
	}

	dbConfig.MaxIdleConnection = getConfigItemUint16Value(configFile, sectionName, "max_idle_conn", defaultDatabaseMaxIdleConn)
	dbConfig.MaxOpenConnection = getConfigItemUint16Value(configFile, sectionName, "max_open_conn", defaultDatabaseMaxOpenConn)
	dbConfig.ConnectionMaxLifeTime = getConfigItemUint32Value(configFile, sectionName, "conn_max_lifetime", defaultDatabaseConnMaxLifetime)

	config.DatabaseConfig = dbConfig
	config.EnableQueryLog = getConfigItemBoolValue(configFile, sectionName, "log_query", false)
	config.AutoUpdateDatabase = getConfigItemBoolValue(configFile, sectionName, "auto_update_database", true)

	return nil
}

func loadMailConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableSMTP = getConfigItemBoolValue(configFile, sectionName, "enable_smtp", false)

	smtpConfig := &SMTPConfig{}
	smtpConfig.SMTPHost = getConfigItemStringValue(configFile, sectionName, "smtp_host")
	smtpConfig.SMTPUser = getConfigItemStringValue(configFile, sectionName, "smtp_user")
	smtpConfig.SMTPPasswd = getConfigItemStringValue(configFile, sectionName, "smtp_passwd")
	smtpConfig.SMTPSkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "smtp_skip_tls_verify", false)

	smtpConfig.FromAddress = getConfigItemStringValue(configFile, sectionName, "from_address")

	config.SMTPConfig = smtpConfig

	return nil
}

func loadLogConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.LogModes = strings.Split(getConfigItemStringValue(configFile, sectionName, "mode", defaultLogMode), " ")

	for i := 0; i < len(config.LogModes); i++ {
		logMode := config.LogModes[i]

		if logMode == "console" {
			config.EnableConsoleLog = true
		} else if logMode == "file" {
			config.EnableFileLog = true
		} else {
			return errs.ErrInvalidLogMode
		}
	}

	var err error
	config.LogLevel, err = getLogLevel(getConfigItemStringValue(configFile, sectionName, "level"))

	if err != nil {
		return err
	}

	if config.LogLevel != LOGLEVEL_DEBUG &&
		config.LogLevel != LOGLEVEL_INFO &&
		config.LogLevel != LOGLEVEL_WARN &&
		config.LogLevel != LOGLEVEL_ERROR {
		return errs.ErrInvalidLogLevel
	}

	if config.LogLevel == LOGLEVEL_DEBUG {
		config.EnableDebugLog = true
	} else {
		config.EnableDebugLog = false
	}

	if config.EnableFileLog {
		fileLogPath := getConfigItemStringValue(configFile, sectionName, "log_path")
		finalFileLogPath, _ := getFinalPath(config.WorkingPath, fileLogPath)
		config.FileLogPath = finalFileLogPath

		requestFileLogPath := getConfigItemStringValue(configFile, sectionName, "request_log_path")

		if requestFileLogPath != "" {
			finalRequestFileLogPath, _ := getFinalPath(config.WorkingPath, requestFileLogPath)
			config.RequestFileLogPath = finalRequestFileLogPath
		} else {
			config.RequestFileLogPath = ""
		}

		queryFileLogPath := getConfigItemStringValue(configFile, sectionName, "query_log_path")

		if queryFileLogPath != "" {
			finalQueryFileLogPath, _ := getFinalPath(config.WorkingPath, queryFileLogPath)
			config.QueryFileLogPath = finalQueryFileLogPath
		} else {
			config.QueryFileLogPath = ""
		}

		config.LogFileRotate = getConfigItemBoolValue(configFile, sectionName, "log_file_rotate", false)
		config.LogFileMaxSize = getConfigItemUint32Value(configFile, sectionName, "log_file_max_size", defaultLogFileMaxSize)
		config.LogFileMaxDays = getConfigItemUint32Value(configFile, sectionName, "log_file_max_days", defaultLogFileMaxDays)
	}

	return nil
}

func loadStorageConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	if getConfigItemStringValue(configFile, sectionName, "type") == LocalFileSystemObjectStorageType {
		config.StorageType = LocalFileSystemObjectStorageType
	} else if getConfigItemStringValue(configFile, sectionName, "type") == S3StorageType {
		config.StorageType = S3StorageType
	} else if getConfigItemStringValue(configFile, sectionName, "type") == MinIOStorageType {
		config.StorageType = MinIOStorageType
	} else if getConfigItemStringValue(configFile, sectionName, "type") == WebDAVStorageType {
		config.StorageType = WebDAVStorageType
	} else {
		return errs.ErrInvalidStorageType
	}

	localFileSystemRootPath := getConfigItemStringValue(configFile, sectionName, "local_filesystem_path")
	finalLocalFileSystemRootPath, err := getFinalPath(config.WorkingPath, localFileSystemRootPath)
	config.LocalFileSystemPath = finalLocalFileSystemRootPath

	if config.StorageType == LocalFileSystemObjectStorageType && err != nil {
		return errs.ErrInvalidLocalFileSystemStoragePath
	}

	s3Config := &S3Config{}
	s3Config.Endpoint = getConfigItemStringValue(configFile, sectionName, "s3_endpoint")
	s3Config.Region = getConfigItemStringValue(configFile, sectionName, "s3_region")
	s3Config.AccessKeyID = getConfigItemStringValue(configFile, sectionName, "s3_access_key_id")
	s3Config.SecretAccessKey = getConfigItemStringValue(configFile, sectionName, "s3_secret_access_key")
	s3Config.SessionToken = getConfigItemStringValue(configFile, sectionName, "s3_session_token")
	s3Config.UseSSL = getConfigItemBoolValue(configFile, sectionName, "s3_use_ssl", false)
	s3Config.SkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "s3_skip_tls_verify", false)
	s3Config.UsePathStyle = getConfigItemBoolValue(configFile, sectionName, "s3_use_path_style", false)
	s3Config.Bucket = getConfigItemStringValue(configFile, sectionName, "s3_bucket")
	s3Config.RootPath = getConfigItemStringValue(configFile, sectionName, "s3_root_path")
	config.S3Config = s3Config

	minIOConfig := &MinIOConfig{}
	minIOConfig.Endpoint = getConfigItemStringValue(configFile, sectionName, "minio_endpoint")
	minIOConfig.Location = getConfigItemStringValue(configFile, sectionName, "minio_location")
	minIOConfig.AccessKeyID = getConfigItemStringValue(configFile, sectionName, "minio_access_key_id")
	minIOConfig.SecretAccessKey = getConfigItemStringValue(configFile, sectionName, "minio_secret_access_key")
	minIOConfig.UseSSL = getConfigItemBoolValue(configFile, sectionName, "minio_use_ssl", false)
	minIOConfig.SkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "minio_skip_tls_verify", false)
	minIOConfig.Bucket = getConfigItemStringValue(configFile, sectionName, "minio_bucket")
	minIOConfig.RootPath = getConfigItemStringValue(configFile, sectionName, "minio_root_path")
	config.MinIOConfig = minIOConfig

	webDAVConfig := &WebDAVConfig{}
	webDAVConfig.Url = getConfigItemStringValue(configFile, sectionName, "webdav_url")
	webDAVConfig.Username = getConfigItemStringValue(configFile, sectionName, "webdav_username")
	webDAVConfig.Password = getConfigItemStringValue(configFile, sectionName, "webdav_password")
	webDAVConfig.RootPath = getConfigItemStringValue(configFile, sectionName, "webdav_root_path")
	webDAVConfig.RequestTimeout = getConfigItemUint32Value(configFile, sectionName, "webdav_request_timeout", defaultWebDAVRequestTimeout)
	webDAVConfig.Proxy = getConfigItemStringValue(configFile, sectionName, "webdav_proxy", "system")
	webDAVConfig.SkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "webdav_skip_tls_verify", false)
	config.WebDAVConfig = webDAVConfig

	return nil
}

func loadAIConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.TransactionFromAITextRecognition = getConfigItemBoolValue(configFile, sectionName, "transaction_from_ai_text_recognition", false)
	config.TransactionFromAIImageRecognition = getConfigItemBoolValue(configFile, sectionName, "transaction_from_ai_image_recognition", true)
	config.InsightsExplorerCodingAssistant = getConfigItemBoolValue(configFile, sectionName, "insights_explorer_coding_assistant", false)
	config.MaxAIRecognitionPictureFileSize = getConfigItemUint32Value(configFile, sectionName, "max_ai_recognition_picture_size", defaultAIRecognitionPictureMaxSize)

	return nil
}

func loadLLMConfiguration(configFile *ini.File, sectionName string, workingPath string, defaultProvider string) (*LLMConfig, error) {
	llmConfig := &LLMConfig{}
	llmProvider := getConfigItemStringValue(configFile, sectionName, "llm_provider", defaultProvider)

	if llmProvider == "" {
		llmConfig.LLMProvider = ""
	} else if llmProvider == OpenAILLMProvider {
		llmConfig.LLMProvider = OpenAILLMProvider
	} else if llmProvider == OpenAICompatibleLLMProvider {
		llmConfig.LLMProvider = OpenAICompatibleLLMProvider
	} else if llmProvider == OpenAIResponsesCompatibleLLMProvider {
		llmConfig.LLMProvider = OpenAIResponsesCompatibleLLMProvider
	} else if llmProvider == AnthropicLLMProvider {
		llmConfig.LLMProvider = AnthropicLLMProvider
	} else if llmProvider == AnthropicCompatibleLLMProvider {
		llmConfig.LLMProvider = AnthropicCompatibleLLMProvider
	} else if llmProvider == OpenRouterLLMProvider {
		llmConfig.LLMProvider = OpenRouterLLMProvider
	} else if llmProvider == OllamaLLMProvider {
		llmConfig.LLMProvider = OllamaLLMProvider
	} else if llmProvider == LMStudioLLMProvider {
		llmConfig.LLMProvider = LMStudioLLMProvider
	} else if llmProvider == GoogleAILLMProvider {
		llmConfig.LLMProvider = GoogleAILLMProvider
	} else if llmProvider == DonutLLMProvider {
		llmConfig.LLMProvider = DonutLLMProvider
	} else {
		return nil, errs.ErrInvalidLLMProvider
	}

	thinkingLevel := getConfigItemStringValue(configFile, sectionName, "enable_thinking")

	if thinkingLevel == string(LLMThinkingDefault) {
		llmConfig.EnableThinking = LLMThinkingDefault
	} else if thinkingLevel == string(LLMThinkingDisabled) {
		llmConfig.EnableThinking = LLMThinkingDisabled
	} else if thinkingLevel == string(LLMThinkingEnabled) {
		llmConfig.EnableThinking = LLMThinkingEnabled
	} else if thinkingLevel == string(LLMThinkingLow) {
		llmConfig.EnableThinking = LLMThinkingLow
	} else if thinkingLevel == string(LLMThinkingMedium) {
		llmConfig.EnableThinking = LLMThinkingMedium
	} else if thinkingLevel == string(LLMThinkingHigh) {
		llmConfig.EnableThinking = LLMThinkingHigh
	} else if thinkingLevel == string(LLMThinkingXHigh) {
		llmConfig.EnableThinking = LLMThinkingXHigh
	} else {
		return nil, errs.ErrInvalidLLMThinkingLevel
	}

	llmConfig.OpenAIAPIKey = getConfigItemStringValue(configFile, sectionName, "openai_api_key")
	llmConfig.OpenAIModelID = getConfigItemStringValue(configFile, sectionName, "openai_model_id")

	llmConfig.OpenAICompatibleBaseURL = getConfigItemStringValue(configFile, sectionName, "openai_compatible_base_url")
	llmConfig.OpenAICompatibleAPIKey = getConfigItemStringValue(configFile, sectionName, "openai_compatible_api_key")
	llmConfig.OpenAICompatibleModelID = getConfigItemStringValue(configFile, sectionName, "openai_compatible_model_id")

	llmConfig.AnthropicAPIKey = getConfigItemStringValue(configFile, sectionName, "anthropic_api_key")
	llmConfig.AnthropicModelID = getConfigItemStringValue(configFile, sectionName, "anthropic_model_id")
	llmConfig.AnthropicMaxTokens = getConfigItemUint32Value(configFile, sectionName, "anthropic_max_tokens", defaultAnthropicLargeLanguageModelAPIMaximumTokens)
	llmConfig.AnthropicThinkingBudgetTokens = getConfigItemUint32Value(configFile, sectionName, "anthropic_thinking_budget_tokens", defaultAnthropicLargeLanguageModelThinkingBudget)

	llmConfig.AnthropicCompatibleBaseURL = getConfigItemStringValue(configFile, sectionName, "anthropic_compatible_base_url")
	llmConfig.AnthropicCompatibleAPIVersion = getConfigItemStringValue(configFile, sectionName, "anthropic_compatible_api_version")
	llmConfig.AnthropicCompatibleAPIKey = getConfigItemStringValue(configFile, sectionName, "anthropic_compatible_api_key")
	llmConfig.AnthropicCompatibleModelID = getConfigItemStringValue(configFile, sectionName, "anthropic_compatible_model_id")
	llmConfig.AnthropicCompatibleMaxTokens = getConfigItemUint32Value(configFile, sectionName, "anthropic_compatible_max_tokens", defaultAnthropicLargeLanguageModelAPIMaximumTokens)
	llmConfig.AnthropicCompatibleThinkingBudgetTokens = getConfigItemUint32Value(configFile, sectionName, "anthropic_compatible_thinking_budget_tokens", defaultAnthropicLargeLanguageModelThinkingBudget)

	llmConfig.OpenRouterAPIKey = getConfigItemStringValue(configFile, sectionName, "openrouter_api_key")
	llmConfig.OpenRouterModelID = getConfigItemStringValue(configFile, sectionName, "openrouter_model_id")

	llmConfig.OllamaServerURL = getConfigItemStringValue(configFile, sectionName, "ollama_server_url")
	llmConfig.OllamaModelID = getConfigItemStringValue(configFile, sectionName, "ollama_model_id")

	llmConfig.LMStudioServerURL = getConfigItemStringValue(configFile, sectionName, "lm_studio_server_url")
	llmConfig.LMStudioToken = getConfigItemStringValue(configFile, sectionName, "lm_studio_token")
	llmConfig.LMStudioModelID = getConfigItemStringValue(configFile, sectionName, "lm_studio_model_id")

	llmConfig.GoogleAIAPIKey = getConfigItemStringValue(configFile, sectionName, "google_ai_api_key")
	llmConfig.GoogleAIModelID = getConfigItemStringValue(configFile, sectionName, "google_ai_model_id")

	llmConfig.DonutModelDownloadURL = getConfigItemStringValue(configFile, sectionName, "donut_model_download_url", defaultDonutModelDownloadURL)
	llmConfig.DonutModelPath, _ = getFinalPath(workingPath, getConfigItemStringValue(configFile, sectionName, "donut_model_path", defaultDonutModelPath))
	llmConfig.DonutWorkerPath = getConfigItemStringValue(configFile, sectionName, "donut_worker_path")
	llmConfig.DonutOnnxRuntimeLibraryPath = getConfigItemStringValue(configFile, sectionName, "donut_onnxruntime_library_path", defaultDonutOnnxRuntimeLibraryPath)
	llmConfig.DonutThreads = getConfigItemUint32Value(configFile, sectionName, "donut_threads", defaultDonutThreads)
	llmConfig.DonutLowMemory = getConfigItemBoolValue(configFile, sectionName, "donut_low_memory", false)
	llmConfig.DonutWorkerIdleTimeout = getConfigItemUint32Value(configFile, sectionName, "donut_worker_idle_timeout", defaultDonutWorkerIdleTimeout)
	llmConfig.DonutDateOrder = strings.ToUpper(getConfigItemStringValue(configFile, sectionName, "donut_date_order", "DMY"))
	llmConfig.DonutDefaultAccount = getConfigItemStringValue(configFile, sectionName, "donut_default_account")

	if llmConfig.DonutDateOrder != "DMY" && llmConfig.DonutDateOrder != "MDY" {
		return nil, errs.ErrInvalidDonutDateOrder
	}

	if llmConfig.DonutThreads < 1 {
		llmConfig.DonutThreads = defaultDonutThreads
	}

	llmConfig.LargeLanguageModelAPIProxy = getConfigItemStringValue(configFile, sectionName, "proxy", "system")
	llmConfig.LargeLanguageModelAPIRequestTimeout = getConfigItemUint32Value(configFile, sectionName, "request_timeout", defaultLargeLanguageModelAPIRequestTimeout)
	llmConfig.LargeLanguageModelAPISkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "skip_tls_verify", false)

	return llmConfig, nil
}

func loadUuidConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	if getConfigItemStringValue(configFile, sectionName, "generator_type") == InternalUuidGeneratorType {
		config.UuidGeneratorType = InternalUuidGeneratorType
	} else {
		return errs.ErrInvalidUuidMode
	}

	config.UuidServerId = getConfigItemUint8Value(configFile, sectionName, "server_id", 0)

	return nil
}

func loadDuplicateCheckerConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	if getConfigItemStringValue(configFile, sectionName, "checker_type") == InMemoryDuplicateCheckerType {
		config.DuplicateCheckerType = InMemoryDuplicateCheckerType
	} else {
		return errs.ErrInvalidDuplicateCheckerType
	}

	config.InMemoryDuplicateCheckerCleanupInterval = getConfigItemUint32Value(configFile, sectionName, "cleanup_interval", defaultInMemoryDuplicateCheckerCleanupInterval)

	if config.InMemoryDuplicateCheckerCleanupInterval < 1 {
		return errs.ErrInvalidInMemoryDuplicateCheckerCleanupInterval
	}

	config.InMemoryDuplicateCheckerCleanupIntervalDuration = time.Duration(config.InMemoryDuplicateCheckerCleanupInterval) * time.Second

	duplicateSubmissionsInterval := getConfigItemUint32Value(configFile, sectionName, "duplicate_submissions_interval", defaultDuplicateSubmissionsInterval)

	config.EnableDuplicateSubmissionsCheck = duplicateSubmissionsInterval > 0

	if duplicateSubmissionsInterval < 1 {
		duplicateSubmissionsInterval = defaultDuplicateSubmissionsInterval
	}

	config.DuplicateSubmissionsInterval = duplicateSubmissionsInterval
	config.DuplicateSubmissionsIntervalDuration = time.Duration(config.DuplicateSubmissionsInterval) * time.Second

	return nil
}

func loadCronConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableRemoveExpiredTokens = getConfigItemBoolValue(configFile, sectionName, "enable_remove_expired_tokens", false)
	config.EnableCreateScheduledTransaction = getConfigItemBoolValue(configFile, sectionName, "enable_create_scheduled_transaction", false)

	return nil
}

func loadBackupConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableScheduledBackup = getConfigItemBoolValue(configFile, sectionName, "enable_scheduled_backup", false)
	config.BackupCron = strings.TrimSpace(getConfigItemStringValue(configFile, sectionName, "backup_cron", defaultBackupCron))

	if _, err := cron.ParseStandard(config.BackupCron); err != nil && config.EnableScheduledBackup {
		return fmt.Errorf("invalid backup cron expression \"%s\": %w", config.BackupCron, err)
	}

	backupPath := strings.TrimSpace(getConfigItemStringValue(configFile, sectionName, "backup_path", defaultBackupPath))
	config.BackupPath, _ = getFinalPath(config.WorkingPath, backupPath)
	config.BackupIncludeConfig = getConfigItemBoolValue(configFile, sectionName, "backup_include_config", true)
	config.BackupEncrypt = getConfigItemBoolValue(configFile, sectionName, "backup_encrypt", false)
	config.BackupPassphrase = strings.TrimRight(getConfigItemStringValue(configFile, sectionName, "backup_passphrase"), "\r\n")
	config.BackupRetentionCount = getConfigItemUint32Value(configFile, sectionName, "backup_retention_count", defaultBackupRetentionCount)
	config.BackupRetentionDays = getConfigItemUint32Value(configFile, sectionName, "backup_retention_days", 0)

	return nil
}

func loadSecurityConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	var err error

	config.SecretKeyNoSet = !getConfigItemIsSet(configFile, sectionName, "secret_key")
	config.SecretKey = getConfigItemStringValue(configFile, sectionName, "secret_key", defaultSecretKey)
	config.TrustedProxyIPs, config.TrustedProxyTextualIPs, err = getCIDRList(configFile, sectionName, "trusted_proxy_ips", defaultTrustedProxyIPs)

	if err != nil {
		return err
	}

	config.TokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "token_expired_time", defaultTokenExpiredTime)

	if config.TokenExpiredTime < 60 {
		return errs.ErrInvalidTokenExpiredTime
	}

	config.TokenExpiredTimeDuration = time.Duration(config.TokenExpiredTime) * time.Second

	config.TokenMinRefreshInterval = getConfigItemUint32Value(configFile, sectionName, "token_min_refresh_interval", defaultTokenMinRefreshInterval)

	if config.TokenMinRefreshInterval >= config.TokenExpiredTime {
		return errs.ErrInvalidTokenMinRefreshInterval
	}

	config.TemporaryTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "temporary_token_expired_time", defaultTemporaryTokenExpiredTime)

	if config.TemporaryTokenExpiredTime < 60 {
		return errs.ErrInvalidTemporaryTokenExpiredTime
	}

	config.TemporaryTokenExpiredTimeDuration = time.Duration(config.TemporaryTokenExpiredTime) * time.Second

	config.EmailVerifyTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "email_verify_token_expired_time", defaultEmailVerifyTokenExpiredTime)

	if config.EmailVerifyTokenExpiredTime < 60 {
		return errs.ErrInvalidEmailVerifyTokenExpiredTime
	}

	config.EmailVerifyTokenExpiredTimeDuration = time.Duration(config.EmailVerifyTokenExpiredTime) * time.Second

	config.PasswordResetTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "password_reset_token_expired_time", defaultPasswordResetTokenExpiredTime)

	if config.PasswordResetTokenExpiredTime < 60 {
		return errs.ErrInvalidPasswordResetTokenExpiredTime
	}

	config.PasswordResetTokenExpiredTimeDuration = time.Duration(config.PasswordResetTokenExpiredTime) * time.Second

	config.UserAvatarTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "user_avatar_token_expired_time", defaultUserAvatarTokenExpiredTime)

	if config.UserAvatarTokenExpiredTime < 60 {
		return errs.ErrInvalidUserAvatarTokenExpiredTime
	}

	config.UserAvatarTokenExpiredTimeDuration = time.Duration(config.UserAvatarTokenExpiredTime) * time.Second

	config.UserCustomIconTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "user_custom_icon_token_expired_time", defaultUserCustomIconTokenExpiredTime)

	if config.UserCustomIconTokenExpiredTime < 60 {
		return errs.ErrInvalidUserCustomIconTokenExpiredTime
	}

	config.UserCustomIconTokenExpiredTimeDuration = time.Duration(config.UserCustomIconTokenExpiredTime) * time.Second

	config.TransactionPictureTokenExpiredTime = getConfigItemUint32Value(configFile, sectionName, "transaction_picture_token_expired_time", defaultTransactionPictureTokenExpiredTime)

	if config.TransactionPictureTokenExpiredTime < 60 {
		return errs.ErrInvalidTransactionPictureTokenExpiredTime
	}

	config.TransactionPictureTokenExpiredTimeDuration = time.Duration(config.TransactionPictureTokenExpiredTime) * time.Second

	config.EnableAPIToken = getConfigItemBoolValue(configFile, sectionName, "enable_api_token", false)
	config.APITokenAllowedRemoteIPs, err = getIPPatterns(configFile, sectionName, "api_token_allowed_remote_ips", "")

	if err != nil {
		return err
	}

	config.MaxFailuresPerIpPerMinute = getConfigItemUint32Value(configFile, sectionName, "max_failures_per_ip_per_minute", defaultMaxFailuresPerIpPerMinute)
	config.MaxFailuresPerUserPerMinute = getConfigItemUint32Value(configFile, sectionName, "max_failures_per_user_per_minute", defaultMaxFailuresPerUserPerMinute)

	return nil
}

func loadAuthConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableInternalAuth = getConfigItemBoolValue(configFile, sectionName, "enable_internal_auth", true)
	config.EnableOAuth2Login = getConfigItemBoolValue(configFile, sectionName, "enable_oauth2_auth", false)
	config.EnableTwoFactor = getConfigItemBoolValue(configFile, sectionName, "enable_two_factor", true)
	config.EnableUserForgetPassword = getConfigItemBoolValue(configFile, sectionName, "enable_forget_password", false)
	config.ForgetPasswordRequireVerifyEmail = getConfigItemBoolValue(configFile, sectionName, "forget_password_require_email_verify", true)
	config.OAuth2ClientID = getConfigItemStringValue(configFile, sectionName, "oauth2_client_id")
	config.OAuth2ClientSecret = getConfigItemStringValue(configFile, sectionName, "oauth2_client_secret")
	config.OAuth2UsePKCE = getConfigItemBoolValue(configFile, sectionName, "oauth2_use_pkce", false)

	oauth2UserIdentifier := getConfigItemStringValue(configFile, sectionName, "oauth2_user_identifier")

	if oauth2UserIdentifier == OAuth2UserIdentifierEmail {
		config.OAuth2UserIdentifier = OAuth2UserIdentifierEmail
	} else if oauth2UserIdentifier == OAuth2UserIdentifierUsername {
		config.OAuth2UserIdentifier = OAuth2UserIdentifierUsername
	} else {
		return errs.ErrInvalidOAuth2UserIdentifier
	}

	config.OAuth2AutoRegister = getConfigItemBoolValue(configFile, sectionName, "oauth2_auto_register", true)

	oauth2Provider := getConfigItemStringValue(configFile, sectionName, "oauth2_provider")

	if oauth2Provider == "" {
		config.OAuth2Provider = ""
	} else if oauth2Provider == OAuth2ProviderOIDC {
		config.OAuth2Provider = OAuth2ProviderOIDC
	} else if oauth2Provider == OAuth2ProviderNextcloud {
		config.OAuth2Provider = OAuth2ProviderNextcloud
	} else if oauth2Provider == OAuth2ProviderGithub {
		config.OAuth2Provider = OAuth2ProviderGithub
	} else {
		return errs.ErrInvalidOAuth2Provider
	}

	config.OAuth2StateExpiredTime = getConfigItemUint32Value(configFile, sectionName, "oauth2_state_expired_time", defaultOAuth2StateExpiredTime)

	if config.OAuth2StateExpiredTime < 60 {
		return errs.ErrInvalidOAuth2StateExpiredTime
	}

	config.OAuth2StateExpiredTimeDuration = time.Duration(config.OAuth2StateExpiredTime) * time.Second

	config.OAuth2Proxy = getConfigItemStringValue(configFile, sectionName, "oauth2_proxy", "system")
	config.OAuth2RequestTimeout = getConfigItemUint32Value(configFile, sectionName, "oauth2_request_timeout", defaultOAuth2RequestTimeout)
	config.OAuth2SkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "oauth2_skip_tls_verify", false)

	config.OAuth2OIDCProviderIssuerURL = getConfigItemStringValue(configFile, sectionName, "oidc_provider_base_url")
	config.OAuth2OIDCProviderCheckIssuerURL = getConfigItemBoolValue(configFile, sectionName, "oidc_provider_check_issuer_url", true)
	config.OAuth2OIDCCustomDisplayNameConfig = getMultiLanguageContentConfig(configFile, sectionName, "enable_oidc_display_name", "oidc_custom_display_name")
	config.OAuth2NextcloudBaseUrl = getConfigItemStringValue(configFile, sectionName, "nextcloud_base_url")

	return nil
}

func loadUserConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableUserRegister = getConfigItemBoolValue(configFile, sectionName, "enable_register", false)
	config.RegisterAllowedEmailDomains = getConfigItemStringListValue(configFile, sectionName, "register_allowed_email_domains")
	config.EnableUserVerifyEmail = getConfigItemBoolValue(configFile, sectionName, "enable_email_verify", false)
	config.EnableUserForceVerifyEmail = getConfigItemBoolValue(configFile, sectionName, "enable_force_email_verify", false)
	config.EnableUserCustomIcon = getConfigItemBoolValue(configFile, sectionName, "enable_custom_icon", false)
	config.MaxUserCustomIconFileSize = getConfigItemUint32Value(configFile, sectionName, "max_user_custom_icon_size", defaultUserCustomIconFileMaxSize)
	config.EnableTransactionPictures = getConfigItemBoolValue(configFile, sectionName, "enable_transaction_picture", false)
	config.MaxTransactionPictureFileSize = getConfigItemUint32Value(configFile, sectionName, "max_transaction_picture_size", defaultTransactionPictureFileMaxSize)
	config.EnableScheduledTransaction = getConfigItemBoolValue(configFile, sectionName, "enable_scheduled_transaction", false)

	if getConfigItemStringValue(configFile, sectionName, "avatar_provider") == string(core.USER_AVATAR_PROVIDER_INTERNAL) {
		config.AvatarProvider = core.USER_AVATAR_PROVIDER_INTERNAL
	} else if getConfigItemStringValue(configFile, sectionName, "avatar_provider") == string(core.USER_AVATAR_PROVIDER_GRAVATAR) {
		config.AvatarProvider = core.USER_AVATAR_PROVIDER_GRAVATAR
	} else if getConfigItemStringValue(configFile, sectionName, "avatar_provider") == "" {
		config.AvatarProvider = ""
	} else {
		return errs.ErrInvalidAvatarProvider
	}

	config.MaxAvatarFileSize = getConfigItemUint32Value(configFile, sectionName, "max_user_avatar_size", defaultUserAvatarFileMaxSize)
	config.DefaultFeatureRestrictions = core.ParseUserFeatureRestrictions(getConfigItemStringValue(configFile, sectionName, "default_feature_restrictions", ""))

	return nil
}

func loadDataConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableDataExport = getConfigItemBoolValue(configFile, sectionName, "enable_export", false)
	config.EnableDataImport = getConfigItemBoolValue(configFile, sectionName, "enable_import", false)
	config.MaxImportFileSize = getConfigItemUint32Value(configFile, sectionName, "max_import_file_size", defaultImportFileMaxSize)

	return nil
}

func loadTipConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.LoginPageTips = getMultiLanguageContentConfig(configFile, sectionName, "enable_tips_in_login_page", "login_page_tips_content")

	return nil
}

func loadNotificationConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	config.AfterRegisterNotification = getMultiLanguageContentConfig(configFile, sectionName, "enable_notification_after_register", "after_register_notification_content")
	config.AfterLoginNotification = getMultiLanguageContentConfig(configFile, sectionName, "enable_notification_after_login", "after_login_notification_content")
	config.AfterOpenNotification = getMultiLanguageContentConfig(configFile, sectionName, "enable_notification_after_open", "after_open_notification_content")

	return nil
}

func loadMapConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	mapProvider := getConfigItemStringValue(configFile, sectionName, "map_provider")

	if mapProvider == "" {
		config.MapProvider = ""
	} else if mapProvider == OpenStreetMapProvider {
		config.MapProvider = OpenStreetMapProvider
	} else if mapProvider == OpenStreetMapHumanitarianStyleProvider {
		config.MapProvider = OpenStreetMapHumanitarianStyleProvider
	} else if mapProvider == OpenTopoMapProvider {
		config.MapProvider = OpenTopoMapProvider
	} else if mapProvider == OPNVKarteMapProvider {
		config.MapProvider = OPNVKarteMapProvider
	} else if mapProvider == CyclOSMMapProvider {
		config.MapProvider = CyclOSMMapProvider
	} else if mapProvider == CartoDBMapProvider {
		config.MapProvider = CartoDBMapProvider
	} else if mapProvider == TomTomMapProvider {
		config.MapProvider = TomTomMapProvider
	} else if mapProvider == TianDiTuProvider {
		config.MapProvider = TianDiTuProvider
	} else if mapProvider == GoogleMapProvider {
		config.MapProvider = GoogleMapProvider
	} else if mapProvider == BaiduMapProvider {
		config.MapProvider = BaiduMapProvider
	} else if mapProvider == AmapProvider {
		config.MapProvider = AmapProvider
	} else if mapProvider == CustomProvider {
		config.MapProvider = CustomProvider
	} else {
		return errs.ErrInvalidMapProvider
	}

	config.EnableMapDataFetchProxy = getConfigItemBoolValue(configFile, sectionName, "map_data_fetch_proxy", false)

	if config.MapProvider == OpenStreetMapProvider ||
		config.MapProvider == OpenStreetMapHumanitarianStyleProvider ||
		config.MapProvider == OpenTopoMapProvider ||
		config.MapProvider == OPNVKarteMapProvider ||
		config.MapProvider == CyclOSMMapProvider ||
		config.MapProvider == CartoDBMapProvider ||
		config.MapProvider == TomTomMapProvider ||
		config.MapProvider == TianDiTuProvider ||
		config.MapProvider == CustomProvider {
		config.SupportMapDataFetchProxy = true
	} else {
		config.SupportMapDataFetchProxy = false
	}

	config.MapProxy = getConfigItemStringValue(configFile, sectionName, "proxy", "system")
	config.TomTomMapAPIKey = getConfigItemStringValue(configFile, sectionName, "tomtom_map_api_key")
	config.TianDiTuAPIKey = getConfigItemStringValue(configFile, sectionName, "tianditu_map_app_key")
	config.GoogleMapAPIKey = getConfigItemStringValue(configFile, sectionName, "google_map_api_key")
	config.BaiduMapAK = getConfigItemStringValue(configFile, sectionName, "baidu_map_ak")
	config.AmapApplicationKey = getConfigItemStringValue(configFile, sectionName, "amap_application_key")

	amapSecurityVerificationMethod := getConfigItemStringValue(configFile, sectionName, "amap_security_verification_method")

	if amapSecurityVerificationMethod == AmapSecurityVerificationInternalProxyMethod {
		config.AmapSecurityVerificationMethod = AmapSecurityVerificationInternalProxyMethod
	} else if amapSecurityVerificationMethod == AmapSecurityVerificationExternalProxyMethod {
		config.AmapSecurityVerificationMethod = AmapSecurityVerificationExternalProxyMethod
	} else if amapSecurityVerificationMethod == AmapSecurityVerificationPlainTextMethod {
		config.AmapSecurityVerificationMethod = AmapSecurityVerificationPlainTextMethod
	} else {
		return errs.ErrInvalidAmapSecurityVerificationMethod
	}

	config.AmapApplicationSecret = getConfigItemStringValue(configFile, sectionName, "amap_application_secret")
	config.AmapApiExternalProxyUrl = getConfigItemStringValue(configFile, sectionName, "amap_api_external_proxy_url")

	config.CustomMapTileServerTileLayerUrl = getConfigItemStringValue(configFile, sectionName, "custom_map_tile_server_url")
	config.CustomMapTileServerAnnotationLayerUrl = getConfigItemStringValue(configFile, sectionName, "custom_map_tile_server_annotation_url")
	config.CustomMapTileServerMinZoomLevel = getConfigItemUint8Value(configFile, sectionName, "custom_map_tile_server_min_zoom_level", 1)
	config.CustomMapTileServerMaxZoomLevel = getConfigItemUint8Value(configFile, sectionName, "custom_map_tile_server_max_zoom_level", 18)
	config.CustomMapTileServerDefaultZoomLevel = getConfigItemUint8Value(configFile, sectionName, "custom_map_tile_server_default_zoom_level", 14)

	return nil
}

func loadExchangeRatesConfiguration(config *Config, configFile *ini.File, sectionName string) error {
	dataSource := getConfigItemStringValue(configFile, sectionName, "data_source")

	if dataSource == CentralBankOfArgentinaDataSource ||
		dataSource == NationalBankOfBelarusDataSource ||
		dataSource == BankOfCanadaDataSource ||
		dataSource == CzechNationalBankDataSource ||
		dataSource == DanmarksNationalbankDataSource ||
		dataSource == EuroCentralBankDataSource ||
		dataSource == NationalBankOfGeorgiaDataSource ||
		dataSource == CentralBankOfHungaryDataSource ||
		dataSource == BankOfIsraelDataSource ||
		dataSource == BankOfItalyDataSource ||
		dataSource == NationalBankOfKazakhstanDataSource ||
		dataSource == CentralBankOfMalaysiaDataSource ||
		dataSource == CentralBankOfMyanmarDataSource ||
		dataSource == NorgesBankDataSource ||
		dataSource == NationalBankOfPolandDataSource ||
		dataSource == NationalBankOfRomaniaDataSource ||
		dataSource == BankOfRussiaDataSource ||
		dataSource == SwissNationalBankDataSource ||
		dataSource == NationalBankOfUkraineDataSource ||
		dataSource == CentralBankOfUzbekistanDataSource ||
		dataSource == UserCustomExchangeRatesDataSource {
		config.ExchangeRatesDataSource = dataSource
	} else {
		return errs.ErrInvalidExchangeRatesDataSource
	}

	config.ExchangeRatesProxy = getConfigItemStringValue(configFile, sectionName, "proxy", "system")
	config.ExchangeRatesRequestTimeout = getConfigItemUint32Value(configFile, sectionName, "request_timeout", defaultExchangeRatesDataRequestTimeout)

	if config.ExchangeRatesRequestTimeout > defaultExchangeRatesDataRequestTimeout {
		config.ExchangeRatesRequestTimeoutExceedDefaultValue = true
	}

	config.ExchangeRatesSkipTLSVerify = getConfigItemBoolValue(configFile, sectionName, "skip_tls_verify", false)

	return nil
}

func loadDeveloperTools(config *Config, configFile *ini.File, sectionName string) error {
	config.EnableLanguagePreview = getConfigItemBoolValue(configFile, sectionName, "enable_language_preview", false)

	return nil
}
