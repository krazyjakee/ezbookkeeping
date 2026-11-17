---
title: "Migrating from ezBookkeeping"
---

# Migrating from ezBookkeeping

ezbookkeeping2 is a business-focused fork of [mayswind/ezbookkeeping](https://github.com/mayswind/ezbookkeeping). It keeps the configuration, data layout and database format of ezBookkeeping, so an existing ezBookkeeping installation can switch to ezbookkeeping2 without migrating any data.

> [!WARNING] Back up first
> Back up your database, the `data` and `storage` directories and your configuration file before switching. Later ezbookkeeping2 releases may add database tables or columns, so switching back to upstream ezBookkeeping afterwards is not guaranteed to work.

## What changes

| Item | ezBookkeeping (upstream) | ezbookkeeping2 |
| --- | --- | --- |
| Product name shown in the app, emails, PWA and docs | `ezBookkeeping` | `ezbookkeeping2` |
| Docker image | `mayswind/ezbookkeeping` | `krazyjakee/ezbookkeeping2` |
| Release archives | `ezbookkeeping-<version>-<os>-<arch>.tar.gz` / `.zip` | `ezbookkeeping2-<version>-<os>-<arch>.tar.gz` / `.zip` |
| Releases | [github.com/mayswind/ezbookkeeping/releases](https://github.com/mayswind/ezbookkeeping/releases) | [github.com/krazyjakee/ezbookkeeping/releases](https://github.com/krazyjakee/ezbookkeeping/releases) |
| Issue tracker | [github.com/mayswind/ezbookkeeping/issues](https://github.com/mayswind/ezbookkeeping/issues) | [github.com/krazyjakee/ezbookkeeping/issues](https://github.com/krazyjakee/ezbookkeeping/issues) |
| Documentation | [ezbookkeeping.mayswind.net](https://ezbookkeeping.mayswind.net) | [krazyjakee.github.io/ezbookkeeping](https://krazyjakee.github.io/ezbookkeeping/) |
| Default names of exported files | `ezBookkeeping_...` | `ezbookkeeping2_...` (the file contents and formats are unchanged) |
| Issuer shown in authenticator apps for newly enabled two-factor authentication | `ezBookkeeping` | `ezbookkeeping2` (existing two-factor setups keep working) |

## Compatibility

The following identifiers stay the same, so existing installations, scripts and integrations keep working:

| Identifier | Value |
| --- | --- |
| Executable name (inside release archives and the Docker image) | `ezbookkeeping` (`ezbookkeeping.exe` on Windows) |
| Configuration file and its sections and keys | `conf/ezbookkeeping.ini` |
| Environment variable prefix | `EBK_` (e.g. `EBK_SERVER_DOMAIN`) |
| Default database names and file paths | e.g. database `ezbookkeeping`, `data/ezbookkeeping.db`, `log/ezbookkeeping.log` |
| Data, storage and log directory layout | `data/`, `storage/`, `log/` |
| Paths inside the Docker container | `/ezbookkeeping/conf`, `/ezbookkeeping/data`, `/ezbookkeeping/storage`, `/ezbookkeeping/log` |
| User and group inside the Docker container | UID / GID `1000` |
| HTTP API, MCP endpoint and OAuth 2.0 callback paths | unchanged, e.g. `/api/v1/...`, `/mcp`, `/oauth2/callback` |
| Browser cookies and local storage keys | unchanged, users stay signed in |
| Import and export data formats | unchanged, files exported from ezBookkeeping can be imported into ezbookkeeping2 and vice versa (shown as "ezBookkeeping File Format" in the import dialog) |
| Go module path | `github.com/mayswind/ezbookkeeping` |
| systemd unit file name | `etc/systemd/ezbookkeeping.service` |
| Application identifier in outgoing HTTP `User-Agent`, API token user agents and the MCP server name | `ezBookkeeping` |

## Migrating a Docker installation

Keep the same volumes, environment variables and configuration file, and only replace the image name `mayswind/ezbookkeeping` with `krazyjakee/ezbookkeeping2`. The image uses the same container paths (`/ezbookkeeping/data`, `/ezbookkeeping/storage`, `/ezbookkeeping/log`, `/ezbookkeeping/conf/ezbookkeeping.ini`) and runs as the same UID / GID `1000`, so no ownership changes are required.

For example, if you started ezBookkeeping with:

```shell
$ docker run -d -p8080:8080 --name ezbookkeeping -v /var/lib/ezbookkeeping/data:/ezbookkeeping/data -v /var/lib/ezbookkeeping/storage:/ezbookkeeping/storage mayswind/ezbookkeeping
```

Stop and remove the old container, then start ezbookkeeping2 with the same volumes:

```shell
$ docker stop ezbookkeeping
$ docker rm ezbookkeeping
$ docker run -d -p8080:8080 --name ezbookkeeping -v /var/lib/ezbookkeeping/data:/ezbookkeeping/data -v /var/lib/ezbookkeeping/storage:/ezbookkeeping/storage krazyjakee/ezbookkeeping2
```

If you use Docker Compose or Kubernetes, change the `image` field to `krazyjakee/ezbookkeeping2` (or a specific version such as `krazyjakee/ezbookkeeping2:1.2.3`) and redeploy.

Available image tags:

* `latest`, `X`, `X.Y` and `X.Y.Z`: release builds
* `latest-snapshot`, `SNAPSHOT-YYYYMMDD` and `sha-<commit>`: daily builds of the `main` branch

## Migrating a binary installation

1. Download the `ezbookkeeping2-<version>-<os>-<arch>` archive for your platform from the [releases page](https://github.com/krazyjakee/ezbookkeeping/releases).
2. Stop the running ezBookkeeping server.
3. Replace the `ezbookkeeping` executable, the `public` and `templates` directories with the ones from the archive. Keep your existing `conf/ezbookkeeping.ini`, `data`, `storage` and `log` directories.
4. Start the server again with the same command, e.g. `./ezbookkeeping server run`. Existing systemd units keep working because the executable name is unchanged.

## After migrating

* Users stay signed in and do not need to change anything. Browsers and installed PWAs pick up the new name after the cached frontend files are refreshed.
* Authenticator apps keep showing the issuer `ezBookkeeping` for two-factor authentication that was enabled before the migration. The codes stay valid.
