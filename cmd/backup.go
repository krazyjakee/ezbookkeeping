package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/mayswind/ezbookkeeping/pkg/backup"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

var backupPassphraseFileFlag = &cli.StringFlag{
	Name:  "passphrase-file",
	Usage: "Read the backup encryption passphrase from the specified file (default is \"backup_passphrase\" in [backup] section, which can also be set by EBK_BACKUP_BACKUP_PASSPHRASE / EBKCFP_BACKUP_BACKUP_PASSPHRASE environment variables)",
}

// Backup represents the backup command
var Backup = &cli.Command{
	Name:  "backup",
	Usage: "ezrbookkeeping full-instance backup and restore",
	Commands: []*cli.Command{
		{
			Name:   "create",
			Usage:  "Create a full-instance backup (database, uploaded files and configuration file)",
			Action: bindBackupAction(createBackup),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Usage:   "Backup output directory (default is \"backup_path\" in [backup] section)",
				},
				&cli.BoolFlag{
					Name:  "no-config",
					Usage: "Do not include the configuration file in the backup",
				},
				&cli.BoolFlag{
					Name:  "encrypt",
					Usage: "Encrypt the backup (default is \"backup_encrypt\" in [backup] section)",
				},
				&cli.BoolFlag{
					Name:  "no-retention",
					Usage: "Do not remove old backups by the retention policy after the backup is created",
				},
				backupPassphraseFileFlag,
			},
		},
		{
			Name:      "list",
			Usage:     "List backups and the status of the latest backup run",
			ArgsUsage: "[backup directory]",
			Action:    bindBackupAction(listBackups),
		},
		{
			Name:      "verify",
			Usage:     "Verify a backup file (checksums, manifest, decryption, data validity and compatibility)",
			ArgsUsage: "<backup file>",
			Action:    bindBackupAction(verifyBackup),
			Flags: []cli.Flag{
				backupPassphraseFileFlag,
			},
		},
		{
			Name:      "restore",
			Usage:     "Restore a backup into the configured database and object storage",
			ArgsUsage: "<backup file>",
			Action:    bindBackupAction(restoreBackup),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Only validate the backup and the target, do not modify anything",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Allow restoring into a non-empty target, ALL existing data in ezrbookkeeping tables will be DELETED (requires --yes-overwrite-existing-data)",
				},
				&cli.BoolFlag{
					Name:  "yes-overwrite-existing-data",
					Usage: "Confirm that existing data may be overwritten (required together with --force)",
				},
				&cli.StringFlag{
					Name:  "config-output",
					Usage: "Write the configuration file contained in the backup to the specified path",
				},
				backupPassphraseFileFlag,
			},
		},
	},
}

type backupActionFunc func(c *core.CliContext, args cli.Args) error

func bindBackupAction(fn backupActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		c := core.WrapCilContext(ctx, cmd)
		return fn(c, cmd.Args())
	}
}

func getBackupPassphrase(c *core.CliContext, config *settings.Config) (string, error) {
	passphraseFile := c.String("passphrase-file")

	if passphraseFile != "" {
		data, err := os.ReadFile(passphraseFile)

		if err != nil {
			return "", fmt.Errorf("cannot read passphrase file: %w", err)
		}

		return strings.TrimRight(string(data), "\r\n"), nil
	}

	return config.BackupPassphrase, nil
}

func createBackup(c *core.CliContext, args cli.Args) error {
	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	options := backup.NewCreateOptionsFromConfig(config)

	if c.String("output") != "" {
		options.OutputDir = c.String("output")
	}

	if c.Bool("no-config") {
		options.IncludeConfig = false
	}

	if c.Bool("encrypt") {
		options.Encrypt = true
	}

	if c.Bool("no-retention") {
		options.ApplyRetention = false
	}

	options.Passphrase, err = getBackupPassphrase(c, config)

	if err != nil {
		return err
	}

	result, err := backup.CreateBackup(c, config, options)

	if err != nil {
		log.CliErrorf(c, "[backup.createBackup] failed to create backup, because %s", err.Error())
		return err
	}

	fmt.Printf("Backup created: %s\n", result.FilePath)
	fmt.Printf("Size: %d bytes\n", result.Size)
	fmt.Printf("Encrypted: %t\n", options.Encrypt)
	fmt.Printf("Configuration file included: %t\n", result.Manifest.ConfigIncluded)
	fmt.Printf("Tables: %d\n", len(result.Manifest.Tables))

	for _, table := range result.Manifest.Tables {
		fmt.Printf("  %s: %d rows\n", table.Name, table.Rows)
	}

	fmt.Printf("Storage objects: %d\n", result.Manifest.StorageObjectCount)

	for _, missingObject := range result.Manifest.MissingStorageObjects {
		fmt.Printf("WARNING: object \"%s\" is referenced by database but missing in object storage\n", missingObject)
	}

	for _, warning := range result.Manifest.Warnings {
		fmt.Printf("WARNING: %s\n", warning)
	}

	for _, deleted := range result.DeletedBackups {
		fmt.Printf("Removed by retention policy: %s\n", deleted)
	}

	for _, retentionError := range result.RetentionErrors {
		fmt.Printf("WARNING: retention policy: %s\n", retentionError)
	}

	return nil
}

func listBackups(c *core.CliContext, args cli.Args) error {
	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	backupDir := config.BackupPath

	if args.Present() {
		backupDir = args.First()
	}

	fmt.Printf("Backup directory: %s\n", backupDir)

	status, err := backup.ReadBackupStatus(backupDir)

	if err != nil {
		log.CliWarnf(c, "[backup.listBackups] cannot read backup status, because %s", err.Error())
	} else if status != nil {
		fmt.Printf("Status:\n")
		fmt.Printf("  Running: %t\n", status.Running)
		fmt.Printf("  Last run started at: %s\n", formatBackupTime(status.LastRunStartedAt))
		fmt.Printf("  Last run finished at: %s\n", formatBackupTime(status.LastRunFinishedAt))
		fmt.Printf("  Last run successful: %t\n", status.LastRunSuccess)
		fmt.Printf("  Last success at: %s (%s)\n", formatBackupTime(status.LastSuccessAt), status.LastSuccessFile)

		if status.LastFailureAt != nil {
			fmt.Printf("  Last failure at: %s\n", formatBackupTime(status.LastFailureAt))
			fmt.Printf("  Last error: %s\n", status.LastError)
		}
	}

	backups, err := backup.ListBackupFiles(backupDir)

	if err != nil {
		log.CliErrorf(c, "[backup.listBackups] cannot list backups, because %s", err.Error())
		return err
	}

	if len(backups) < 1 {
		fmt.Printf("No backups found\n")
		return nil
	}

	fmt.Printf("Backups (newest first):\n")

	for _, backupFile := range backups {
		fmt.Printf("  %s  created at %s  %d bytes  encrypted: %t\n", backupFile.Name, backupFile.CreatedAt.Format(time.RFC3339), backupFile.Size, backupFile.Encrypted)
	}

	return nil
}

func verifyBackup(c *core.CliContext, args cli.Args) error {
	if !args.Present() {
		return errors.New("backup file is required")
	}

	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	passphrase, err := getBackupPassphrase(c, config)

	if err != nil {
		return err
	}

	report, err := backup.VerifyBackup(c, args.First(), passphrase)

	if report != nil {
		printVerifyReport(report)
	}

	if err != nil {
		log.CliErrorf(c, "[backup.verifyBackup] backup verification failed, because %s", err.Error())
		return err
	}

	fmt.Printf("Backup is valid\n")

	return nil
}

func restoreBackup(c *core.CliContext, args cli.Args) error {
	if !args.Present() {
		return errors.New("backup file is required")
	}

	config, err := initializeSystem(c)

	if err != nil {
		return err
	}

	passphrase, err := getBackupPassphrase(c, config)

	if err != nil {
		return err
	}

	options := &backup.RestoreOptions{
		Passphrase:       passphrase,
		DryRun:           c.Bool("dry-run"),
		Force:            c.Bool("force"),
		ConfirmOverwrite: c.Bool("yes-overwrite-existing-data"),
		ConfigOutputPath: c.String("config-output"),
	}

	if options.Force && !options.ConfirmOverwrite && !options.DryRun {
		return errors.New("--force requires --yes-overwrite-existing-data to confirm that ALL existing data will be deleted")
	}

	report, err := backup.RestoreBackup(c, config, args.First(), options)

	if report != nil {
		printRestoreReport(report)
	}

	if err != nil {
		log.CliErrorf(c, "[backup.restoreBackup] restore failed, because %s", err.Error())
		return err
	}

	if options.DryRun {
		fmt.Printf("Dry run finished, nothing has been modified\n")
	} else {
		fmt.Printf("Restore finished successfully\n")
	}

	return nil
}

func printVerifyReport(report *backup.VerifyReport) {
	fmt.Printf("Backup file: %s\n", report.FilePath)
	fmt.Printf("Encrypted: %t\n", report.Encrypted)

	if manifest := report.Manifest; manifest != nil {
		fmt.Printf("Format version: %d\n", manifest.FormatVersion)
		fmt.Printf("Created at: %s\n", manifest.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Application version: %s (commit %s)\n", valueOrUnknown(manifest.AppVersion), valueOrUnknown(manifest.AppCommitHash))
		fmt.Printf("Database type: %s\n", manifest.DatabaseType)
		fmt.Printf("Storage type: %s\n", manifest.StorageType)
		fmt.Printf("Configuration file included: %t\n", manifest.ConfigIncluded)
		fmt.Printf("Tables:\n")

		for _, table := range manifest.Tables {
			fmt.Printf("  %s: %d rows\n", table.Name, table.Rows)
		}

		fmt.Printf("Storage objects: %d\n", manifest.StorageObjectCount)
	}

	for _, warning := range report.Warnings {
		fmt.Printf("WARNING: %s\n", warning)
	}

	for _, problem := range report.Problems {
		fmt.Printf("ERROR: %s\n", problem)
	}
}

func printRestoreReport(report *backup.RestoreReport) {
	if report.Verify != nil {
		printVerifyReport(report.Verify)
	}

	if report.Verify == nil || !report.Verify.OK() {
		return
	}

	fmt.Printf("Target is empty: %t\n", report.TargetEmpty)

	for _, table := range report.NonEmptyTables {
		fmt.Printf("  existing data: %s\n", table)
	}

	if report.ConflictingObjects > 0 {
		fmt.Printf("  existing storage objects with same path: %d\n", report.ConflictingObjects)
	}

	if report.DryRun {
		fmt.Printf("Restore plan:\n")

		for _, table := range report.Tables {
			fmt.Printf("  %s: %d rows to restore (%d existing rows)\n", table.Name, table.BackupRows, table.ExistingRows)
		}

		fmt.Printf("  storage objects to restore: %d\n", report.StorageObjects)
	} else if len(report.Highlights) > 0 {
		fmt.Printf("Verification report:\n")

		for _, table := range report.Tables {
			digest := "content identical"

			if !table.DigestMatch {
				digest = table.DigestNote
			}

			fmt.Printf("  table %s: %d restored / %d in backup (%s)\n", table.Name, table.RestoredRows, table.BackupRows, digest)
		}

		fmt.Printf("  users loadable: %d / %d\n", report.UsersLoaded, report.UsersTotal)

		for _, highlight := range report.Highlights {
			fmt.Printf("  %s\n", highlight)
		}
	}

	if report.ConfigWrittenTo != "" {
		fmt.Printf("Configuration file written to: %s\n", report.ConfigWrittenTo)
	}

	for _, warning := range report.Warnings {
		fmt.Printf("WARNING: %s\n", warning)
	}

	for _, problem := range report.Problems {
		fmt.Printf("ERROR: %s\n", problem)
	}
}

func formatBackupTime(t *time.Time) string {
	if t == nil {
		return "never"
	}

	return t.Format(time.RFC3339)
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}

	return value
}
