---
title: Developing
---

# Developing

## Preparation

ezrbookkeeping is written in Golang and TypeScript. You should install [Git](https://git-scm.com/), [Golang](https://golang.org/), [GCC](https://gcc.gnu.org/), [Node.js](https://nodejs.org/) and [NPM](https://www.npmjs.com/) before you start to develop. The release package and docker image of ezrbookkeeping is compiled in Golang 1.27 and Node.js 26.x. The Go versions required by previous ezrbookkeeping releases can be found in [System Requirements](/installation/system_requirements).

## Download the source code

You can use `git clone` to obtain the latest source code.

```shell
$ git clone https://github.com/krazyjakee/ezbookkeeping.git
```

## Develop and debug

The `main` function of backend is in file `/ezbookkeeping.go`. You can specifiy this file to build and run, then ezbookkeeping will listen on port `8080` by default. The other backend files are mostly in the directory `pkg`.

The entry of frontend page is `/src/index-main.ts`, and the homepage will redirect to the desktop or mobile page depending on the visitor's device type. The desktop frontend entry is `/src/desktop-main.ts`, and the mobile frontend entry is `/src/mobile-main.ts`. Almost all frontend files are in the same directory `src`. You can use `npm run serve` to compile frontend files and start a web server which listens on port `8081` and proxy backend request to port `8080`.

After doing the above steps, you can open the url `http://127.0.0.1:8081/` in your browser, and then you can develop and debug your code.

## Building

You can use `build.sh` / `build.bat` / `build.ps1` to build the frontend, backend or full package. `build.sh` also supports building a Docker image.

## Verification

From the repository root, install the locked frontend dependencies and run the checks separately:

```shell
npm ci
npm run lint
npm test
npm run build
go vet ./...
go test ./...
go build .
```

`npm run lint` checks TypeScript and ESLint without changing files. Use `npm run lint:fix` to apply ESLint fixes explicitly.

Some Go tests call live third-party APIs. To use the CI mode that omits those calls, run `BUILD_PIPELINE=1 go test ./...` on a POSIX shell. Set `CHECK_3RD_API=1` as well to enable them in CI. On PowerShell, set these variables with `$env:BUILD_PIPELINE = "1"` and `$env:CHECK_3RD_API = "1"` before running the Go tests.

The build scripts run lint and tests by default. `SKIP_TESTS` is a Go test-name regular expression passed to `go test -skip`; it is not a boolean. To omit all tests, use the script's `--no-test` or `-NoTest` option.

## Build targets

### Building frontend files

**Linux / macOS**

```shell
$ ./build.sh frontend
```

**Windows**

```shell
> .\build.bat frontend
```

or

```powershell
PS > .\build.ps1 frontend
```

The frontend files will be built and be placed in the `dist` directory.

### Building backend file

**Linux / macOS**

```shell
$ ./build.sh backend
```

**Windows**

```shell
> .\build.bat backend
```

or

```powershell
PS > .\build.ps1 backend
```

The backend file will be built and be placed in the project directory and the file name is `ezbookkeeping`.

### Building full package

**Linux / macOS**

```shell
$ ./build.sh package -o ezrbookkeeping.tar.gz
```

All the files will be built and be packaged in `ezrbookkeeping.tar.gz`.

**Windows**

```shell
> .\build.bat package -o ezrbookkeeping.zip
```

or

```powershell
PS > .\build.ps1 package -Output ezrbookkeeping.zip
```

All the files will be packaged in `ezrbookkeeping.zip`.

### Building docker image

**Linux**

```shell
$ ./build.sh docker -t ezrbookkeeping:release
```

The docker image will be built and be tagged with `ezrbookkeeping:release`.
