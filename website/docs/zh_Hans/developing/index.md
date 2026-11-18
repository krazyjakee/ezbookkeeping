---
title: 开发
---

# 开发

## 准备

ezrbookkeeping 使用 Golang 和 TypeScript 编写。在您开始开发前，你需要安装 [Git](https://git-scm.com/)、[Golang](https://golang.org/)、[GCC](https://gcc.gnu.org/)、[Node.js](https://nodejs.org/) 和 [NPM](https://www.npmjs.com/)。ezrbookkeeping 的发布包和 Docker 镜像使用 Golang 1.27 和 Node.js 26.x 进行编译。ezrbookkeeping 之前版本依赖的 Golang 版本可以在 [系统要求](/zh_Hans/installation/system_requirements) 中找到。

## 下载源代码

您可以使用 `git clone` 获取最近源代码。

```shell
$ git clone https://github.com/krazyjakee/ezbookkeeping.git
```

## 开发和调试

后端的 `main` 函数在文件 `/ezbookkeeping.go` 中。你可以指定该文件进行编译和运行，然后 ezbookkeeping 默认会监听端口 `8080`。后端其他的文件大部分在目录 `pkg` 中。

前端页面的入口是 `/src/index-main.ts`，首页将会根据访问者设备的类型跳转到桌面版或移动版的网页。桌面版的前端入口是 `/src/desktop-main.ts`，移动版的前端入口是 `/src/mobile-main.ts`。几乎所有前端文件都在相同的目录 `src` 下。你可以使用 `npm run serve` 编译前端文件并启动一个 Web 服务器，监听端口 `8081`，并将后端请求代理到端口 `8080`。

做完上述步骤后，你可以在你的浏览器中打开网址 `http://127.0.0.1:8081/`，然后你就可以开发并调试你的代码了。

## 编译

您可以使用 `build.sh` / `build.bat` / `build.ps1` 脚本构建前端文件、后端文件和完整包。`build.sh` 还支持构建 Docker 镜像。

## 验证

在仓库根目录安装锁定版本的前端依赖，然后分别运行检查：

```shell
npm ci
npm run lint
npm test
npm run build
go vet ./...
go test ./...
go build .
```

`npm run lint` 检查 TypeScript 和 ESLint，不会修改文件。若要应用 ESLint 自动修复，请显式运行 `npm run lint:fix`。

部分 Go 测试会请求第三方在线 API。在 POSIX shell 中，可以运行 `BUILD_PIPELINE=1 go test ./...`，使用省略这些请求的 CI 模式。在 CI 中同时设置 `CHECK_3RD_API=1` 可启用这些测试。在 PowerShell 中，请先使用 `$env:BUILD_PIPELINE = "1"` 和 `$env:CHECK_3RD_API = "1"` 设置相应变量，再运行 Go 测试。

构建脚本默认执行静态检查和测试。`SKIP_TESTS` 是传递给 `go test -skip` 的测试名称正则表达式，并非布尔值。若要省略全部测试，请使用脚本的 `--no-test` 或 `-NoTest` 选项。

## 构建目标

### 构建前端文件

**Linux / macOS**

```shell
$ ./build.sh frontend
```

**Windows**

```shell
> .\build.bat frontend
```

或

```powershell
PS > .\build.ps1 frontend
```

前端文件将会构建并放置到 `dist` 目录中。

### 构建后端文件

**Linux / macOS**

```shell
$ ./build.sh backend
```

**Windows**

```shell
> .\build.bat backend
```

或

```powershell
PS > .\build.ps1 backend
```

后端文件将会构建并放置到项目目录中，文件名为 `ezbookkeeping`。

### 构建完整包

**Linux / macOS**

```shell
$ ./build.sh package -o ezrbookkeeping.tar.gz
```

所有文件将会构建并打包到 `ezrbookkeeping.tar.gz`。

**Windows**

```shell
> .\build.bat package -o ezrbookkeeping.zip
```

或

```powershell
PS > .\build.ps1 package -Output ezrbookkeeping.zip
```

所有文件将会构建并打包到 `ezrbookkeeping.zip`。

### 构建 Docker 镜像

**Linux**

```shell
$ ./build.sh docker -t ezrbookkeeping:release
```

Docker 镜像将会构建并设置标签为 `ezrbookkeeping:release`。
