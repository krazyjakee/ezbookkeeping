---
layout: home

hero:
    eyebrow: "自托管 · 企业记账"
    name: "ezrbookkeeping"
    text: "稳定、面向企业的自托管记账应用"
    tagline: "mayswind/ezbookkeeping 的商业化分支。稳定性、成熟度与商业用途优先，不包含个人生活记录功能。"
    actions:
        - theme: brand
          text: 快速开始
          link: /zh_Hans/installation
        - theme: alt
          text: 在 GitHub 上查看
          link: https://github.com/krazyjakee/ezbookkeeping

screenshots:
    - light: /images/screenshots/desktop_overview_light.jpg
      dark: /images/screenshots/desktop_overview_dark.jpg
      label: 桌面版 · 概览
      alt: ezrbookkeeping 桌面版概览
      kind: desktop
    - light: /images/screenshots/mobile_overview_light.jpg
      dark: /images/screenshots/mobile_overview_dark.jpg
      label: 移动版 · 概览
      alt: ezrbookkeeping 移动版概览
      kind: mobile

features:
    - title: 开源 & 自托管
      icon: "01 · 托管"
      details:
          专为隐私与数据自主而设计
    - title: 轻量 & 快速
      icon: "02 · 资源占用"
      details:
          资源占用极少，适合家庭服务器、NAS 以及树莓派 (Raspberry Pi)
    - title: 安装简单
      icon: "03 · 部署"
      details:
          支持 Docker <br/>
          支持 SQLite、MySQL、PostgreSQL 多种数据库 <br/>
          跨平台运行 (Windows, macOS, Linux, FreeBSD) <br/>
          支持 x86, amd64, ARM 架构
    - title: 友好的用户界面
      icon: "04 · 界面"
      details:
          针对手机与桌面优化的 UI <br/>
          支持 PWA，带来接近原生 App 的使用体验 <br/>
          自定义首页布局 <br/>
          深色模式
    - title: AI 驱动的功能
      icon: "05 · AI"
      details:
          文字、收据图片识别 <br/>
          AI 自定义图表生成 <br/>
          MCP (Model Context Protocol) 用于 AI 集成 <br/>
          Agent Skill 和 API 命令行脚本工具用于 AI 集成
    - title: 强大的记账功能
      icon: "06 · 记账"
      details:
          两级账户与两级分类 <br/>
          支持交易图片附件 <br/>
          记录交易地理位置并在地图上展示 <br/>
          支持定时交易 <br/>
          高级筛选、搜索、数据可视化与分析功能
    - title: 本地化与国际化
      icon: "07 · 本地化"
      details:
          多语言与多币种支持 <br/>
          多汇率数据源及汇率自动更新 <br/>
          多时区支持 <br/>
          自定义日期、数字与货币格式
    - title: 安全可靠
      icon: "08 · 安全"
      details:
          两步认证 (2FA) <br/>
          OIDC 外部认证<br/>
          登录频次限制 <br/>
          应用锁 (PIN 码 / WebAuthn)
    - title: 数据导入 & 导出
      icon: "09 · 数据迁移"
      details:
          导入 CSV、Excel、OFX、QFX、QIF、IIF、Camt.052、Camt.053、MT940、GnuCash、Firefly III、Beancount、随手记，以及支付宝、微信支付及京东金融的对账单 等多种格式 <br/>
          CSV、Excel 导入支持自定义列映射、规则以及脚本代码

sitemap:
    - name: 文档
      links:
        - name: 安装指南
          url: /zh_Hans/installation/
        - name: 基础使用
          url: /zh_Hans/basic_usage/
        - name: 导出 & 导入
          url: /zh_Hans/export_and_import/
        - name: 与 OpenClaw 集成
          url: /zh_Hans/agent/openclaw
    - name: 资源
      links:
        - name: 功能列表
          url: /zh_Hans/features/
          external: false
        - name: 更新日志
          url: https://github.com/krazyjakee/ezbookkeeping/releases
          external: true
        - name: Docker 仓库
          url: https://hub.docker.com/r/krazyjakee/ezrbookkeeping
          external: true
    - name: 帮助
      links:
        - name: 常见问题
          url: /zh_Hans/faq/
        - name: 功能对比
          url: /zh_Hans/comparison/
        - name: GitHub 问题反馈
          url: https://github.com/krazyjakee/ezbookkeeping/issues
          external: true
---
