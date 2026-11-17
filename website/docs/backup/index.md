---
title: Backup and Restore
---

# Backup and Restore

ezrbookkeeping includes built-in full-instance backups. A backup is a single archive file which can be used to restore a complete installation onto a clean instance, including:

* **Database**: every row of every table managed by ezrbookkeeping (users, two-factor settings, sessions, accounts and balances, transactions, categories, tags, templates and scheduled transactions, pictures / attachments metadata, custom icons, import presets, custom exchange rates, application settings, external auth bindings, insights explorers, transaction references, MCP records).
* **Uploaded files**: every object in the configured object storage (user avatars, transaction pictures and attachments including PDFs, user custom icons).
* **Configuration file** (optional, enabled by default): the configuration file as it exists on disk (e.g. `conf/ezbookkeeping.ini`).

Backups work with every supported database (SQLite, MySQL / MariaDB, PostgreSQL) and every supported object storage (`local_filesystem`, `s3`, `minio`, `webdav`). No external tools (`mysqldump`, `pg_dump`, `mc`, ...) are required, because ezrbookkeeping reads and writes the data itself through the same database and object storage connections it already uses.

## What is NOT included

> [!WARNING] ATTENTION
> Configuration values supplied by **environment variables** (`EBK_{SECTION}_{OPTION}`) or **files referenced by environment variables** (`EBKCFP_{SECTION}_{OPTION}`, e.g. Docker / Kubernetes secrets) are **not** captured, because they are not part of the configuration file. Keep them in your secret manager, `docker-compose.yml`, Kubernetes manifests or password manager, and make sure you can recreate them during a restore.

The most important of these values is `secret_key` in the [security](/configuration/#security) section. Two-factor authentication secrets and session tokens are encrypted / signed with it. **Restore with the same `secret_key` as the original instance**, otherwise users with two-factor authentication cannot log in and all existing sessions become invalid.

Also not included:

* The database server itself (users, permissions, server settings) and the object storage bucket / WebDAV share settings. Create an empty database and bucket before restoring.
* Log files, TLS certificates and key files referenced by the configuration (`cert_file`, `cert_key_file`), the static frontend files and templates (they are part of the release package / Docker image).
* Map tiles, exchange rate data and other data fetched from third parties at runtime.

## Backup file format

Backups are written as `ezrbookkeeping-backup-<UTC timestamp>.tar.gz`, or `ezrbookkeeping-backup-<UTC timestamp>.tar.gz.enc` when encrypted. The archive contains:

| Entry | Description |
| --- | --- |
| `database/<table>.jsonl` | All the rows of a table, one JSON object per line, ordered by primary key. |
| `storage/<avatar\|transaction\|icon>/<path>` | All the objects in object storage. |
| `config/ezbookkeeping.ini` | The configuration file (when included). |
| `manifest.json` | Metadata: format version, application version and commit, creation time, database type, storage type, row count and columns of every table, size and SHA-256 checksum of every entry, objects referenced by the database but missing in object storage, and warnings. |

The manifest is written last, so an interrupted backup never has a valid manifest. In-progress backups are written as `*.partial` files and only renamed after the complete archive has been written, synced to disk and verified, so every file named `ezrbookkeeping-backup-*.tar.gz[.enc]` is a completed backup.

Encrypted backups use AES-256-GCM in chunked streaming mode with a key derived from the passphrase via scrypt. Every chunk is authenticated, so wrong passphrases, modified, truncated and appended data are all detected.

## Consistency

All the database tables are read inside **one read transaction** (a `REPEATABLE READ` read-only snapshot on MySQL / MariaDB and PostgreSQL, a read transaction holding a shared lock on SQLite), so the database part of a backup is always a consistent point-in-time snapshot. Object storage files are copied after the database snapshot.

* Files uploaded while a backup is running may be included without being referenced by the database. This is harmless, they are just unused files.
* Files deleted while a backup is running may be missing. Objects which are referenced by the database but missing in object storage are recorded in the manifest and reported as warnings by `backup create`, `backup verify` and `backup restore`.
* On SQLite, writes are blocked or may fail while the database tables are being read (usually well under a second for typical installations). Schedule backups at a quiet time.

For a strictly consistent recovery point (database and files at exactly the same moment), stop the web server (or block user access) during the backup, i.e. use a maintenance window.

## Creating backups manually

```shell
# write a backup into "backup_path" (default: data/backups) using the [backup] settings
./ezbookkeeping backup create

# write into another directory, encrypt, and exclude the configuration file
EBK_BACKUP_BACKUP_PASSPHRASE='a long passphrase' ./ezbookkeeping backup create --output /mnt/backups --encrypt --no-config

# read the passphrase from a file
./ezbookkeeping backup create --encrypt --passphrase-file /run/secrets/ezbookkeeping_backup_passphrase
```

Options of `backup create`:

* `--output value`, `-o value`: Backup output directory. Optional. (default is `backup_path` in the `[backup]` section)
* `--no-config`: Do not include the configuration file. Optional.
* `--encrypt`: Encrypt the backup. Optional. (default is `backup_encrypt` in the `[backup]` section)
* `--no-retention`: Do not apply the retention policy after the backup. Optional.
* `--passphrase-file value`: Read the passphrase from a file. Optional. (default is `backup_passphrase`)

Run `ezbookkeeping database update` first if the database structure is outdated (the web server does this automatically on start when `auto_update_database` is enabled).

## Listing and verifying backups

```shell
# list backups (newest first) and the status of the latest backup run
./ezbookkeeping backup list
./ezbookkeeping backup list /mnt/backups

# verify a backup without restoring it
./ezbookkeeping backup verify data/backups/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
```

`backup verify` checks that the file can be decrypted, the archive is complete (not truncated) and not corrupted, the manifest exists and matches the archive, the size and SHA-256 checksum of every entry, that every table file contains valid JSON rows matching the row count in the manifest, and that the backup is compatible with the running version (see [compatibility](#compatibility)). It exits with a non-zero status if any check fails.

## Scheduled backups

Scheduled backups run inside the web server process (`ezbookkeeping server run`). Configure them in the `[backup]` section (see [Configuration - Backup](/configuration/#backup)):

```ini
[backup]
enable_scheduled_backup = true
backup_cron = 0 3 * * *
backup_path = data/backups
backup_include_config = true
backup_encrypt = true
backup_retention_count = 7
backup_retention_days = 30
```

* `backup_cron` uses the standard 5-field cron format (`minute hour day-of-month month day-of-week`) in the server's local time zone (set `TZ` for Docker containers).
* Overlapping runs are prevented by an in-process lock and a lock file (`.ezrbookkeeping-backup.lock`) in the backup directory. A lock file left by a crashed process is detected and removed automatically.
* Every run is logged (`[backup.CreateBackup] ... created successfully` or `failed to create backup, because ...`), and the result is written to `backup-status.json` in the backup directory (last run time, last success time and file, last failure time and error). `backup list` prints this status. Monitor this file or the log for failures.
* You can trigger the scheduled job manually with `ezbookkeeping cron run --name ScheduledBackup`.

### Retention

After a **successful** backup, older backups in the backup directory exceeding `backup_retention_count` (keep at most N backups) or `backup_retention_days` (keep backups at most N days old) are deleted. `0` disables the respective limit.

* Retention is never applied after a failed backup, so failures never remove the last known good backup.
* The newest successful backup is never deleted, even if it is older than `backup_retention_days`.
* Only completed backup files named `ezrbookkeeping-backup-*` are considered; other files in the directory are left untouched.

Manual `backup create` runs also apply the retention policy unless `--no-retention` is set.

## Restoring

> [!WARNING] ATTENTION
> Stop the ezrbookkeeping web server before restoring.

1. Install the **same or a newer** version of ezrbookkeeping on the target host. A backup created by a newer version cannot be restored by an older version.
2. Prepare the configuration: use the configuration file from the backup (extract it with `--config-output`, see below, or keep your own copy), recreate all the values supplied by environment variables / secret files, and use the **same `secret_key`**. The database and object storage settings may point to new, empty locations.
3. Create the empty database (MySQL / PostgreSQL) and object storage bucket / directory.
4. Validate the backup and the target without changing anything:

    ```shell
    ./ezbookkeeping backup restore --dry-run /path/to/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
    ```

5. Restore:

    ```shell
    ./ezbookkeeping backup restore /path/to/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
    ```

6. Start the web server and log in.

Options of `backup restore`:

* `--dry-run`: Only validate the backup and the target, nothing is modified. Optional.
* `--force`: Allow restoring into a non-empty target. **All existing rows of all ezrbookkeeping tables are deleted**, objects with the same path are overwritten. Requires `--yes-overwrite-existing-data`. Optional.
* `--yes-overwrite-existing-data`: Explicit confirmation required together with `--force`. Optional.
* `--config-output value`: Write the configuration file contained in the backup to this path (mode `0600`). Optional.
* `--passphrase-file value`: Read the passphrase from a file. Optional. (default is `backup_passphrase`)

The restore process:

1. Fully verifies the backup (the same checks as `backup verify`) **before** any modification.
2. Checks that the target database tables and object storage are empty (objects with the same paths as in the backup count as existing data). If not, it stops unless `--force --yes-overwrite-existing-data` are given.
3. Creates / updates the database structure (the same as `database update`).
4. Inserts the rows of each table in batches inside one transaction per table, preserving all primary keys, and uploads all the objects.
5. Prints a verification report: row count and content checksum of every table compared to the backup, whether every user can be loaded, counts of accounts (including balances), transactions, categories, tags, templates and scheduled transactions, application settings, pictures and custom icons, and the checksum of every restored object.

If a restore fails in the middle, the tables restored before the failure stay committed. Fix the problem, clean the target (or use `--force --yes-overwrite-existing-data`) and run the restore again.

The restore uses the database and object storage configured for the running instance, so it can also restore a backup into a **different database type or object storage type** (e.g. a backup of an SQLite instance into PostgreSQL). In that case the content checksum comparison may report differences in value representation as warnings; the row counts are still verified.

### Compatibility

* Backups created by an older version of ezrbookkeeping can be restored: tables and columns added in newer versions are created by the database structure update and keep their default (empty) values.
* Backups created by a newer version are refused (newer application version, newer backup format version, unknown tables or unknown columns).
* If the application version is unknown (e.g. local development builds), compatibility is checked by tables and columns only, and a warning is printed.

## Docker

Mount a dedicated volume for backups and keep secrets in environment variables or Docker secrets:

```yaml
services:
  ezbookkeeping:
    image: krazyjakee/ezrbookkeeping:latest
    environment:
      - TZ=Europe/London
      - EBK_BACKUP_ENABLE_SCHEDULED_BACKUP=true
      - EBK_BACKUP_BACKUP_CRON=0 3 * * *
      - EBK_BACKUP_BACKUP_PATH=/ezbookkeeping/backups
      - EBK_BACKUP_BACKUP_ENCRYPT=true
      - EBK_BACKUP_BACKUP_RETENTION_COUNT=14
      - EBKCFP_BACKUP_BACKUP_PASSPHRASE=/run/secrets/backup_passphrase
      - EBKCFP_SECURITY_SECRET_KEY=/run/secrets/secret_key
    secrets:
      - backup_passphrase
      - secret_key
    volumes:
      - ./data:/ezbookkeeping/data
      - ./storage:/ezbookkeeping/storage
      - ./backups:/ezbookkeeping/backups
secrets:
  backup_passphrase:
    file: ./secrets/backup_passphrase
  secret_key:
    file: ./secrets/secret_key
```

The container runs as uid `1000`, so the backup directory on the host must be writable by uid `1000` (`sudo chown 1000:1000 ./backups`).

Create, list and verify backups inside the running container:

```shell
docker exec ezbookkeeping /ezbookkeeping/ezbookkeeping backup create
docker exec ezbookkeeping /ezbookkeeping/ezbookkeeping backup list
docker exec ezbookkeeping /ezbookkeeping/ezbookkeeping backup verify /ezbookkeeping/backups/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
```

Restore into a new, stopped installation by running a one-off container with the same volumes and environment variables:

```shell
docker compose stop ezbookkeeping
docker compose run --rm ezbookkeeping /ezbookkeeping/ezbookkeeping backup restore --dry-run /ezbookkeeping/backups/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
docker compose run --rm ezbookkeeping /ezbookkeeping/ezbookkeeping backup restore /ezbookkeeping/backups/ezrbookkeeping-backup-20261117T030000.000Z.tar.gz.enc
docker compose start ezbookkeeping
```

## Binary installation

Either enable scheduled backups in the `[backup]` section (backups run while `ezbookkeeping server run` is running), or keep `enable_scheduled_backup = false` and use a systemd timer (or cron) to run `backup create`, which applies the retention policy too:

```ini
# /etc/systemd/system/ezbookkeeping-backup.service
[Unit]
Description=ezrbookkeeping backup

[Service]
Type=oneshot
User=ezbookkeeping
WorkingDirectory=/opt/ezbookkeeping
Environment=EBKCFP_BACKUP_BACKUP_PASSPHRASE=/etc/ezbookkeeping/backup_passphrase
ExecStart=/opt/ezbookkeeping/ezbookkeeping backup create
```

```ini
# /etc/systemd/system/ezbookkeeping-backup.timer
[Unit]
Description=Daily ezrbookkeeping backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

```shell
sudo systemctl enable --now ezbookkeeping-backup.timer
systemctl list-timers ezbookkeeping-backup.timer
journalctl -u ezbookkeeping-backup.service
```

A failed backup makes the service fail, so it can be picked up by your monitoring (e.g. `OnFailure=` units).

## Protecting backups

Backups contain all financial data, password hashes, two-factor secrets (encrypted with `secret_key`) and, when the configuration file is included, database / storage credentials.

* Backup files are created with mode `0600` and the backup directory with mode `0700`. Keep the backup directory readable only by the ezrbookkeeping user and your backup tooling.
* Enable `backup_encrypt` before copying backups to another host or to cloud storage, and transfer them over encrypted channels (e.g. `rsync` over SSH, `rclone` with HTTPS).
* Supply `backup_passphrase` via `EBK_BACKUP_BACKUP_PASSPHRASE` or `EBKCFP_BACKUP_BACKUP_PASSPHRASE` instead of writing it in the configuration file (the configuration file itself is included in backups). The passphrase is never logged or written into backups.
* Store the passphrase and `secret_key` separately from the backups, e.g. in a password manager. **Without the passphrase, an encrypted backup cannot be restored.**
* Keep at least one copy of your backups off the host (3-2-1 rule): `backup_path` on the same disk as the database does not protect against disk failure.

## Periodic restore drill

A backup that has never been restored is not a backup. Regularly (e.g. monthly) and after every upgrade:

1. Run `ezbookkeeping backup verify` on the latest backup (this can be automated, e.g. right after the backup timer).
2. Restore the latest backup into a temporary instance: a separate directory / container with its own empty SQLite database (or an empty MySQL / PostgreSQL database) and its own empty `local_filesystem` storage path, using the production `secret_key`:

    ```shell
    mkdir -p /tmp/ezb-drill/conf /tmp/ezb-drill/data /tmp/ezb-drill/storage
    cp conf/ezbookkeeping.ini /tmp/ezb-drill/conf/
    # point the drill instance at its own database and storage
    export EBK_DATABASE_TYPE=sqlite3 EBK_DATABASE_DB_PATH=/tmp/ezb-drill/data/ezbookkeeping.db
    export EBK_STORAGE_TYPE=local_filesystem EBK_STORAGE_LOCAL_FILESYSTEM_PATH=/tmp/ezb-drill/storage
    ./ezbookkeeping --conf-path /tmp/ezb-drill/conf/ezbookkeeping.ini backup restore /path/to/latest-backup.tar.gz.enc
    ```

3. Check the verification report (all tables `content identical` or matching row counts, all users loadable, all storage objects verified).
4. Optionally start the drill instance on another port (`EBK_SERVER_HTTP_PORT`), log in, and spot-check account balances, recent transactions, scheduled transactions and attachments.
5. Delete the drill instance afterwards, it contains real data.
