---
title: 与 OpenClaw 集成
---

# 与 OpenClaw 集成

[OpenClaw](https://openclaw.ai/) 是一个在你自己设备上运行的个人 AI 助手，其支持对接多种常见的通道（如 WhatsApp、Telegram、Slack 等）以及对接常见的大语言模型提供方（如 OpenAI、Anthropic、Google AI 等），实现用户通过自然语言让 AI Agent 在自己的设备上完成各项任务。

ezrbookkeeping 为 AI Agent 提供了通过 HTTP API 操作 ezrbookkeeping 的 [Agent 技能](/zh_Hans/agent/skill)，这个技能也适用于 OpenClaw，OpenClaw 可以调用这个技能在 ezrbookkeeping 执行记账、查询等任务。

## 安装 OpenClaw

请参考 [安装 - OpenClaw](https://docs.openclaw.ai/install) 进行安装。

> [!CAUTION] 警告
> 请谨慎使用第三方的大语言模型服务，因为你的隐私数据会发送给第三方大模型提供方。

## 安装 ezrbookkeeping 技能

ezrbookkeeping 适配 OpenClaw 的技能已经发布在 [ClawHub](https://clawhub.ai)。

1. 您可以直接通过 ClawHub 的命令行工具安装：

```shell
npx clawhub@latest install ezbookkeeping
```

2. 或者可以访问如下地址下载：

[https://clawhub.ai/mayswind/ezbookkeeping](https://clawhub.ai/mayswind/ezbookkeeping)

## 配置 ezrbookkeeping API Tools

与 MCP (Model Context Protocol) 不同，ezrbookkeeping API Tools 使用的是 API 令牌，您需要先在 ezrbookkeeping 中创建一个 API 令牌，具体请参考 [API - 使用](/zh_Hans/httpapi/#使用)。

然后您可以在您的用户主目录下创建一个 `.env` 文件，包含 `EBKTOOL_SERVER_BASEURL` ezrbookkeeping 服务器地址和 `EBKTOOL_TOKEN` API 令牌两个环境变量，例如：

```
EBKTOOL_SERVER_BASEURL="https://{YOUR_HOST_ADDRESS}"
EBKTOOL_TOKEN="{YOUR-API-TOKEN}"
```

如果您没有主动配置这些环境变量，与 OpenClaw 对话时，OpenClaw 也会提示您输入这些信息。ezrbookkeeping API Tools 使用的更多说明，请参考 [API - API 工具脚本](/zh_Hans/httpapi/#api-工具脚本) 文档。
