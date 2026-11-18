---
title: "Installation"
---

# Introduction

ezrbookkeeping is an open source, powerful, self-hosted personal finance app that is easy to use. It helps you record daily transactions, import data from various sources, and quickly search and filter your bills. You can analyze historical data using built-in charts or perform custom queries with your own chart dimensions to better understand spending patterns and financial trends. ezrbookkeeping is easy to deploy, and you can start it with just one single Docker command. Designed to be resource-efficient, it runs smoothly on devices such as Raspberry Pi, NAS, and MicroServers.

ezrbookkeeping offers tailored interfaces for both mobile and desktop devices. With support for PWA (Progressive Web Apps), you can even [add it to your mobile home screen](/faq/#how-to-add-ezrbookkeeping-to-home-screen-of-mobile-phone) and use it like a native app.

ezrbookkeeping uses a lightweight tech stack and an efficient architecture to keep the application minimal, while still providing powerful features. You can explore all supported features in [Feature List](/features/).

## Run with Docker
Visit [Docker Hub](https://hub.docker.com/r/krazyjakee/ezrbookkeeping) to see all images and tags.

```shell
$ docker run -p8080:8080 krazyjakee/ezrbookkeeping:latest
```

> [!NOTE] Note
> The above commands are intended for testing and evaluation purposes only. For production environments, it is recommended to use a persistent data volume to store data to avoid data loss.

## Install from Binary
Download the latest release: [https://github.com/krazyjakee/ezbookkeeping/releases](https://github.com/krazyjakee/ezbookkeeping/releases)

**Linux / macOS / FreeBSD**

```shell
$ ./ezbookkeeping server run
```

**Windows**

```shell
> .\ezbookkeeping.exe server run
```

By default, ezrbookkeeping listens on port 8080. You can then visit `http://{YOUR_HOST_ADDRESS}:8080/` .

## Build from Source
Make sure you have [Golang](https://golang.org/), [GCC](https://gcc.gnu.org/), [Node.js](https://nodejs.org/) and [NPM](https://www.npmjs.com/) installed. Then download the source code, and follow these steps:

**Linux / macOS / FreeBSD**

```shell
$ ./build.sh package -o ezrbookkeeping.tar.gz
```

All the files will be packaged in `ezrbookkeeping.tar.gz`.

**Windows**

```shell
> .\build.bat package -o ezrbookkeeping.zip
```

or

```powershell
PS > .\build.ps1 package -Output ezrbookkeeping.zip
```

All the files will be packaged in `ezrbookkeeping.zip`.

You can also build a Docker image. Make sure you have [Docker](https://www.docker.com/) installed, then follow these steps:

**Linux**

```shell
$ ./build.sh docker
```
