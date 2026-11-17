---
layout: home

hero:
    eyebrow: "Self-hosted · Business bookkeeping"
    name: "ezrbookkeeping"
    text: "Stable, self-hosted bookkeeping for businesses"
    tagline: "A business-focused fork of mayswind/ezbookkeeping. Stability, maturity and business use come first. Personal life logging is out of scope."
    actions:
        - theme: brand
          text: Get Started
          link: /installation
        - theme: alt
          text: View on GitHub
          link: https://github.com/krazyjakee/ezbookkeeping

screenshots:
    - light: /images/screenshots/desktop_overview_light.jpg
      dark: /images/screenshots/desktop_overview_dark.jpg
      label: Desktop · Overview
      alt: ezrbookkeeping desktop overview
      kind: desktop
    - light: /images/screenshots/mobile_overview_light.jpg
      dark: /images/screenshots/mobile_overview_dark.jpg
      label: Mobile · Overview
      alt: ezrbookkeeping mobile overview
      kind: mobile

features:
    - title: Open Source & Self-Hosted
      icon: "01 · Hosting"
      details:
          Built for privacy and control
    - title: Lightweight & Fast
      icon: "02 · Footprint"
      details:
          Minimal resource usage, suitable for home servers, NAS and Raspberry Pi
    - title: Easy Installation
      icon: "03 · Deployment"
      details:
          Docker support <br/>
          Supports SQLite, MySQL, PostgreSQL <br/>
          Cross-platform (Windows, macOS, Linux, FreeBSD) <br/>
          Works on x86, amd64, ARM architectures
    - title: User-Friendly Interface
      icon: "04 · Interface"
      details:
          UI optimized for both mobile and desktop <br/>
          PWA support for native-like mobile experience <br/>
          Customizable overview layouts <br/>
          Dark mode
    - title: AI-Powered Features
      icon: "05 · AI"
      details:
          Text and receipt image recognition <br/>
          AI custom chart generation <br/>
          MCP (Model Context Protocol) for AI integration <br/>
          Agent Skill and API command-line script tools for AI integration
    - title: Powerful Bookkeeping
      icon: "06 · Bookkeeping"
      details:
          Two-level accounts and categories <br/>
          Image attachments for transactions <br/>
          Location tracking with maps <br/>
          Scheduled transactions <br/>
          Advanced filtering, search, visualization and analysis
    - title: Localization & Internationalization
      icon: "07 · Localization"
      details:
          Multi-language and multi-currency support <br/>
          Multiple exchange rate sources with automatic updates <br/>
          Multi-timezone support <br/>
          Custom formats for dates, numbers and currencies
    - title: Security
      icon: "08 · Security"
      details:
          Two-factor authentication (2FA) <br/>
          OIDC external authentication <br/>
          Login rate limiting <br/>
          Application lock (PIN code / WebAuthn)
    - title: Data Import & Export
      icon: "09 · Data transfer"
      details:
          Import CSV, Excel, OFX, QFX, QIF, IIF, Camt.052, Camt.053, MT940, GnuCash, Firefly III, Beancount and more <br/>
          Custom column mapping, rules and scripts for CSV / Excel imports

sitemap:
    - name: Docs
      links:
        - name: Installation
          url: /installation/
        - name: Basic Usage
          url: /basic_usage/
        - name: Export & Import
          url: /export_and_import/
        - name: Integration with OpenClaw
          url: /agent/openclaw
    - name: Resources
      links:
        - name: Feature List
          url: /features/
          external: false
        - name: Release Notes
          url: https://github.com/krazyjakee/ezbookkeeping/releases
          external: true
        - name: Docker Repository
          url: https://hub.docker.com/r/krazyjakee/ezrbookkeeping
          external: true
    - name: Help
      links:
        - name: FAQ
          url: /faq/
        - name: Comparison
          url: /comparison/
        - name: GitHub Issues
          url: https://github.com/krazyjakee/ezbookkeeping/issues
          external: true
---
