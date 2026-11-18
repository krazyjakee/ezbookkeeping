package cron

import (
	"fmt"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/backup"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// RemoveExpiredTokensJob represents the cron job which periodically remove expired user tokens from the database
var RemoveExpiredTokensJob = &CronJob{
	Name:        "RemoveExpiredTokens",
	Description: "Periodically remove expired user tokens from the database.",
	Period: CronJobFixedHourPeriod{
		Hour: 0,
	},
	Run: func(c *core.CronContext) error {
		return services.Tokens.DeleteAllExpiredTokens(c)
	},
}

// CreateScheduledTransactionJob represents the cron job which periodically create transaction by scheduled transaction template
var CreateScheduledTransactionJob = &CronJob{
	Name:        "CreateScheduledTransaction",
	Description: "Periodically create transaction by scheduled transaction template.",
	Period: CronJobEvery15MinutesPeriod{
		Second: 0,
	},
	Run: func(c *core.CronContext) error {
		return services.Transactions.CreateScheduledTransactions(c, time.Now().Unix(), c.GetInterval())
	},
}

// ScheduledBackupJobName is the name of the scheduled full-instance backup cron job
const ScheduledBackupJobName = "ScheduledBackup"

// NewScheduledBackupJob returns the cron job which periodically creates full-instance backups according to the [backup] section of the config
func NewScheduledBackupJob(config *settings.Config) *CronJob {
	return &CronJob{
		Name:        ScheduledBackupJobName,
		Description: fmt.Sprintf("Periodically create full-instance backups into \"%s\" and apply the retention policy.", config.BackupPath),
		Period: CronJobCronExpressionPeriod{
			Expression: config.BackupCron,
		},
		Run: func(c *core.CronContext) error {
			_, err := backup.CreateBackup(c, config, backup.NewCreateOptionsFromConfig(config))
			return err
		},
	}
}
