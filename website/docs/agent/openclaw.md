---
title: Integration with OpenClaw
---

# Integration with OpenClaw

[OpenClaw](https://openclaw.ai/) is a personal AI assistant you run on your own devices. It supports integration with multiple common channels (such as WhatsApp, Telegram, Slack, etc.) and major large language model providers (such as OpenAI, Anthropic, Google AI, etc.), allowing users to use natural language to have AI agents perform various tasks locally.

ezbookkeeping2 provides an [Agent Skill](/agent/skill) that allows AI agents to operate ezbookkeeping2 through the HTTP API. This skill is also compatible with OpenClaw, which can invoke to perform tasks such as recording transactions and querying data in ezbookkeeping2.

## Install OpenClaw

Please refer to [Install - OpenClaw](https://docs.openclaw.ai/install) for installation instructions.

> [!CAUTION] Warning
> Use third-party large language model services with caution, as your private data will be sent to external model providers.

## Install ezbookkeeping2 Skill

The OpenClaw-compatible skill for ezbookkeeping2 is available on [ClawHub](https://clawhub.ai).

1. You can install it directly using the ClawHub CLI:

```shell
npx clawhub@latest install ezbookkeeping
```

2. Or download it from the following page:

[https://clawhub.ai/mayswind/ezbookkeeping](https://clawhub.ai/mayswind/ezbookkeeping)

## Configure ezbookkeeping2 API Tools

Unlike MCP (Model Context Protocol), ezbookkeeping2 API Tools use an API token. You must first create an API token in ezbookkeeping2. For details, see [API - Usage](/httpapi/#usage).

You can create a `.env` file in your home directory that defines the two environment variables `EBKTOOL_SERVER_BASEURL` for the ezbookkeeping2 server address and `EBKTOOL_TOKEN` for the API token, for example:

```
EBKTOOL_SERVER_BASEURL="https://{YOUR_HOST_ADDRESS}"
EBKTOOL_TOKEN="{YOUR-API-TOKEN}"
```

If you do not configure these environment variables in advance, OpenClaw will prompt you to enter them during the conversation. For more details on using the ezbookkeeping2 API Tools, refer to the [API - API Tool Script](/httpapi/#api-tool-script) documentation.
