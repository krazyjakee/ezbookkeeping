# ezrbookkeeping

> **A business-focused fork of [mayswind/ezbookkeeping](https://github.com/mayswind/ezbookkeeping).**
> Upstream is moving toward personal life logging: experiences, memories and photographs. This fork sticks to bookkeeping and puts these first:
> - **Stability:** conservative changes, correctness and data integrity before new features
> - **Maturity:** predictable releases and safe upgrades for long-running deployments
> - **Business use:** features for small businesses and teams, not personal lifestyle tracking

[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/krazyjakee/ezbookkeeping/blob/main/LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/krazyjakee/ezbookkeeping/build.yml?branch=main)](https://github.com/krazyjakee/ezbookkeeping/actions/workflows/build.yml)
[![Latest Docker Image Size](https://img.shields.io/docker/image-size/krazyjakee/ezrbookkeeping.svg?style=flat)](https://hub.docker.com/r/krazyjakee/ezrbookkeeping)
[![Docker Pulls](https://img.shields.io/docker/pulls/krazyjakee/ezrbookkeeping)](https://hub.docker.com/r/krazyjakee/ezrbookkeeping)
[![Documentation](https://img.shields.io/badge/docs-krazyjakee.github.io-blue.svg)](https://krazyjakee.github.io/ezbookkeeping/)
[![Forked from mayswind/ezbookkeeping](https://img.shields.io/badge/forked%20from-mayswind%2Fezbookkeeping-lightgrey.svg?logo=github)](https://github.com/mayswind/ezbookkeeping)

## Introduction
ezrbookkeeping is a self-hosted bookkeeping app for recording transactions, managing accounts and categories, and importing financial data. Built-in charts and custom queries help you review balances, spending and income. It supports SQLite, MySQL and PostgreSQL, with Docker images and standalone binaries for deployment.

ezrbookkeeping offers tailored interfaces for mobile and desktop devices. With PWA (Progressive Web Apps) support, you can add it to your mobile home screen (open it in your phone's browser, then choose **Add to Home Screen** or **Install app** from the browser menu) and use it like a native app.

## Features
- **Open Source & Self-Hosted**
    - Built for privacy and control
- **Lightweight & Fast**
    - Minimal resource usage, suitable for home servers, NAS and Raspberry Pi
- **Easy Installation**
    - Docker support
    - SQLite, MySQL and PostgreSQL support
    - Cross-platform (Windows, macOS, Linux, FreeBSD)
    - Works on x86, amd64, ARM architectures
- **User-Friendly Interface**
    - UI optimized for both mobile and desktop
    - PWA support for native-like mobile experience
    - Customizable overview layouts
    - Dark mode
- **AI-Powered Features**
    - Text and receipt image recognition
    - AI custom chart generation
    - MCP (Model Context Protocol) for AI integration
    - Agent Skill and API command-line script tools for AI integration
- **Powerful Bookkeeping**
    - Two-level accounts and categories
    - Image attachments for transactions
    - Location tracking with maps
    - Scheduled transactions
    - Advanced filtering, search, visualization and analysis
- **Localization & Internationalization**
    - Multi-language and multi-currency support
    - Multiple exchange rate sources with automatic updates
    - Multi-timezone support
    - Custom formats for dates, numbers and currencies
- **Security**
    - Two-factor authentication (2FA)
    - OIDC external authentication
    - Login rate limiting
    - Application lock (PIN code / WebAuthn)
- **Data Import & Export**
    - Import CSV, Excel, OFX, QFX, QIF, IIF, Camt.052, Camt.053, MT940, GnuCash, Firefly III, Beancount and more
    - Custom column mapping, rules and scripts for CSV / Excel imports

For a full list of features, visit the [Full Feature List](https://krazyjakee.github.io/ezbookkeeping/features/).

## Screenshots
### Desktop Version
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/desktop_overview_dark.jpg">
  <img src="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/desktop_overview_light.jpg" alt="ezrbookkeeping desktop overview with monthly position, asset summary, period tiles, expense trends and transaction calendar">
</picture>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/desktop_transactions_dark.jpg">
  <img src="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/desktop_transactions_light.jpg" alt="ezrbookkeeping desktop transaction list grouped by date with signed amounts">
</picture>

### Mobile Version
<p>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/mobile_overview_dark.jpg">
  <img src="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/mobile_overview_light.jpg" alt="ezrbookkeeping mobile overview with monthly position and period summary" width="292">
</picture>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/mobile_transaction_add_dark.jpg">
  <img src="https://raw.githubusercontent.com/krazyjakee/ezbookkeeping/main/website/docs/public/images/screenshots/mobile_transaction_add_light.jpg" alt="ezrbookkeeping mobile transaction entry" width="292">
</picture>
</p>

Screenshots use the bundled demo data and follow your light / dark preference. See the [documentation site](https://krazyjakee.github.io/ezbookkeeping/) for more.

## Installation
### Run with Docker
Visit [Docker Hub](https://hub.docker.com/r/krazyjakee/ezrbookkeeping) to see all images and tags.

    $ docker run -p8080:8080 krazyjakee/ezrbookkeeping:latest

For production use of ezrbookkeeping, you need to mount a persistent volume when starting the container to prevent data loss. For more information, see [documentation](https://krazyjakee.github.io/ezbookkeeping/installation/installation-docker).

There is one image tag, `latest`, rebuilt on every push to `main`.

#### Migrating from mayswind/ezbookkeeping
Just change the image from `mayswind/ezbookkeeping` to `krazyjakee/ezrbookkeeping` and keep everything else: the same volumes (`/ezbookkeeping/data`, `/ezbookkeeping/storage`, `/ezbookkeeping/log`), the same UID/GID `1000`, the same configuration file (`/ezbookkeeping/conf/ezbookkeeping.ini`) and the same `EBK_` environment variables. No data migration is needed. Back up your data first, see [Migrating from ezBookkeeping](https://krazyjakee.github.io/ezbookkeeping/installation/migrating) for details.

### Install from Binary
Download the latest release: [https://github.com/krazyjakee/ezbookkeeping/releases](https://github.com/krazyjakee/ezbookkeeping/releases)

Release archives are named `ezrbookkeeping-<version>-<os>-<arch>`; the executable inside is still named `ezbookkeeping`.

**Linux / macOS / FreeBSD**

    $ ./ezbookkeeping server run

**Windows**

    > .\ezbookkeeping.exe server run

By default, ezrbookkeeping listens on port 8080. You can then visit `http://{YOUR_HOST_ADDRESS}:8080/` .

### Build from Source
Make sure you have [Golang](https://golang.org/), [GCC](https://gcc.gnu.org/), [Node.js](https://nodejs.org/) and [NPM](https://www.npmjs.com/) installed. Then download the source code, and follow these steps:

**Linux / macOS / FreeBSD**

    $ ./build.sh package -o ezrbookkeeping.tar.gz

All the files will be packaged in `ezrbookkeeping.tar.gz`.

**Windows**

    > .\build.bat package -o ezrbookkeeping.zip

or

    PS > .\build.ps1 package -Output ezrbookkeeping.zip

All the files will be packaged in `ezrbookkeeping.zip`.

You can also build a Docker image. Make sure you have [Docker](https://www.docker.com/) installed, then follow these steps:

**Linux**

    $ ./build.sh docker

## Contributing
We welcome contributions of all kinds.

See the [development guide](https://krazyjakee.github.io/ezbookkeeping/developing/) for the application entrypoints, local setup, build commands and verification checks. `npm run lint` checks types and lint rules without editing files; use `npm run lint:fix` to apply lint fixes explicitly.

If you find a bug, please [submit an issue](https://github.com/krazyjakee/ezbookkeeping/issues) on GitHub.

If you would like to contribute code, you can fork the repository and open a pull request.

Improvements to documentation, feature suggestions, and other forms of feedback are also appreciated.

You can view existing contributors on the [Contributor Graph](https://github.com/krazyjakee/ezbookkeeping/graphs/contributors). ezrbookkeeping builds on the work of [MaysWind](https://github.com/mayswind) and all [upstream ezBookkeeping contributors](https://github.com/mayswind/ezbookkeeping/graphs/contributors).

## Translating
Help make ezrbookkeeping accessible to users around the world. We welcome help to improve existing translations or add new ones. If you would like to contribute a translation, please refer to the [translation guide](https://krazyjakee.github.io/ezbookkeeping/translating).

Currently available translations:

| Tag | Language | Progress | Contributors |
| --- | --- | --- | --- |
| de | Deutsch | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fde.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/de.json) | [@chrgm](https://github.com/chrgm), [@1270o1](https://github.com/1270o1), [@martinschilliger](https://github.com/martinschilliger), [@marcelweikum](https://github.com/marcelweikum) |
| el | Ελληνικά | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fel.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/el.json) | [@konet-gr](https://github.com/konet-gr) |
| en | English | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fen.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/en.json) | / |
| es | Español | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fes.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/es.json) | [@Miguelonlonlon](https://github.com/Miguelonlonlon), [@abrugues](https://github.com/abrugues), [@AndresTeller](https://github.com/AndresTeller), [@diegofercri](https://github.com/diegofercri) |
| fr | Français | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Ffr.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/fr.json) | [@brieucdlf](https://github.com/brieucdlf) |
| hi | हिन्दी | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fhi.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/hi.json) | [@rahul05ranjan](https://github.com/rahul05ranjan) |
| it | Italiano | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fit.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/it.json) | [@waron97](https://github.com/waron97), [@albanobattistella](https://github.com/albanobattistella), [@mcagl](https://github.com/mcagl) |
| ja | 日本語 | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fja.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/ja.json) | [@tkymmm](https://github.com/tkymmm), [@Mink16](https://github.com/Mink16), [@x0x0b](https://github.com/x0x0b) |
| kn | ಕನ್ನಡ | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fkn.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/kn.json) | [@Darshanbm05](https://github.com/Darshanbm05) |
| ko | 한국어 | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fko.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/ko.json) | [@overworks](https://github.com/overworks) |
| nl | Nederlands | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fnl.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/nl.json) | [@automagics](https://github.com/automagics) |
| pl | Polski | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fpl.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/pl.json) | [@mnowok](https://github.com/mnowok) |
| pt-BR | Português (Brasil) | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fpt-BR.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/pt-BR.json) | [@thecodergus](https://github.com/thecodergus), [@balaios](https://github.com/balaios) |
| ro | Română | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fro.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/ro.json) | [@gg64nou](https://github.com/gg64nou) |
| ru | Русский | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fru.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/ru.json) | [@artegoser](https://github.com/artegoser), [@dshemin](https://github.com/dshemin), [@zhugaru](https://github.com/zhugaru) |
| sk | Slovenčina | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fsk.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/sk.json) | [@Laybr1990](https://github.com/Laybr1990) |
| sl | Slovenščina | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fsl.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/sl.json) | [@thehijacker](https://github.com/thehijacker) |
| ta | தமிழ் | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fta.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/ta.json) | [@hhharsha36](https://github.com/hhharsha36) |
| th | ไทย | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fth.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/th.json) | [@natthavat28](https://github.com/natthavat28) |
| tr | Türkçe | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Ftr.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/tr.json) | [@aydnykn](https://github.com/aydnykn), [@snizamaddinov](https://github.com/snizamaddinov) |
| uk | Українська | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fuk.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/uk.json) | [@nktlitvinenko](https://github.com/nktlitvinenko), [@grid-pilot](https://github.com/grid-pilot), [@infinit1ve](https://github.com/infinit1ve) |
| vi | Tiếng Việt | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fvi.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/vi.json) | [@f97](https://github.com/f97) |
| zh-Hans | 中文 (简体) | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fzh-Hans.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/zh-Hans.json) | / |
| zh-Hant | 中文 (繁體) | [![Translation Progress](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fmayswind%2FezBookkeeping-i18n-badge%2Fmain%2Fbadges%2Fzh-Hant.json)](https://github.com/mayswind/ezBookkeeping-i18n-badge/blob/main/untranslated/zh-Hant.json) | / |

## Documentation
1. [English](https://krazyjakee.github.io/ezbookkeeping)
1. [中文 (简体)](https://krazyjakee.github.io/ezbookkeeping/zh_Hans)

## Compatibility
ezrbookkeeping uses its own product name, Docker image (`krazyjakee/ezrbookkeeping`) and release archive names (`ezrbookkeeping-<version>-<os>-<arch>`). The following identifiers are intentionally retained for compatibility with existing installations:

- Executable name `ezbookkeeping` (inside release archives and the Docker image)
- Configuration file `conf/ezbookkeeping.ini` and all of its keys, and the `EBK_` environment variable prefix
- Data, storage and log directory layout and default database names (e.g. `data/ezbookkeeping.db`)
- Docker container paths (`/ezbookkeeping/...`) and UID/GID `1000`
- HTTP API, MCP and OAuth 2.0 callback paths, cookies and browser local storage keys
- Import and export data formats
- Go module path `github.com/mayswind/ezbookkeeping`

See [Migrating from ezBookkeeping](https://krazyjakee.github.io/ezbookkeeping/installation/migrating) for the full list.

## License
[MIT](https://github.com/krazyjakee/ezbookkeeping/blob/main/LICENSE)

ezrbookkeeping is based on [ezBookkeeping](https://github.com/mayswind/ezbookkeeping), Copyright (c) 2020-2026 MaysWind.
