---
title: Agent Skill
---

# Agent Skill

[Agent Skills](https://agentskills.io) are a lightweight, open format for extending AI agent capabilities with specialized knowledge and workflows.

Since AI agents are well suited to using command line tools, ezrbookkeeping provides **ezrbookkeeping API Tools**, which wrap the HTTP API as a command line interface to simplify how AI agents interact with ezrbookkeeping APIs. The default output of ezrbookkeeping API Tools is designed to be as compact as possible while preserving semantics, in order to reduce token consumption when used with large language models.

In addition, ezrbookkeeping provides a [Agent Skill](https://github.com/krazyjakee/ezbookkeeping/tree/main/skills/ezbookkeeping) for using ezrbookkeeping API Tools. AI agents can directly use this skill to perform tasks such as recording transactions and querying data.

## Install Skill

1. You can install it directly using the [Skills](https://skills.sh/) CLI:

```shell
npx skills add krazyjakee/ezbookkeeping@ezbookkeeping
```

2. Or download it from the following page:

The ezrbookkeeping skill consists of two parts: the SKILL.md file and the corresponding ezrbookkeeping API Tools command line utility. You can download both components directly from the GitHub repository.

[https://github.com/krazyjakee/ezbookkeeping/blob/main/skills/ezbookkeeping](https://github.com/krazyjakee/ezbookkeeping/blob/main/skills/ezbookkeeping)

The directory structure after downloading is as follows:

```
.
├── ezbookkeeping
│   ├── SKILL.md
│   └── scripts
│       ├── ebktools.ps1
│       └── ebktools.sh
```

You can then install the `ezbookkeeping` directory into the appropriate skills directory of the platform where your AI agent.

> [!CAUTION] Warning
> Use third-party large language model services with caution, as your private data will be sent to external model providers.

## Configure ezrbookkeeping API Tools

Unlike MCP (Model Context Protocol), ezrbookkeeping API Tools use an API token. You must first create an API token in ezrbookkeeping. For details, see [API - Usage](/httpapi/#usage).

You can create a `.env` file in the `scripts` directory mentioned above, in the ezrbookkeeping skill directory, or in your user home directory that defines the two environment variables `EBKTOOL_SERVER_BASEURL` for the ezrbookkeeping server address and `EBKTOOL_TOKEN` for the API token, for example:

```
EBKTOOL_SERVER_BASEURL="https://{YOUR_HOST_ADDRESS}"
EBKTOOL_TOKEN="{YOUR-API-TOKEN}"
```

After completing the configuration, the AI agent can call the ezrbookkeeping API through the skill described above. For more details on how to use ezrbookkeeping API Tools, refer to the [API - API Tool Script](/httpapi/#api-tool-script) documentation.
