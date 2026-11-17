import { type HeadConfig, defineConfig, resolveSiteDataByRoute } from 'vitepress';

const websiteName: string = 'ezbookkeeping2';
const websiteBasePath: string = '/ezbookkeeping/';
const websiteUrlRootUrl: string = 'https://krazyjakee.github.io/ezbookkeeping';
const repositoryUrl: string = 'https://github.com/krazyjakee/ezbookkeeping';
const titleEn: string = 'ezbookkeeping2 - stable, self-hosted bookkeeping for businesses';
const titleZhHans: string = 'ezbookkeeping2 - 稳定、面向企业的自托管 (self-hosted) 记账应用';
const descriptionEn: string = 'ezbookkeeping2 is a stable, business-focused, self-hosted bookkeeping app. A fork of mayswind/ezbookkeeping focused on maturity and business use.';
const descriptionZhHans: string = 'ezbookkeeping2 是一款稳定、面向企业的自托管 (self-hosted) 记账应用，是 mayswind/ezbookkeeping 的分支，专注于成熟度与商业用途。';

// https://vitepress.dev/reference/site-config
export default defineConfig({
    title: titleEn,
    description: descriptionEn,
    base: websiteBasePath,
    srcDir: './docs',
    cleanUrls: true,
    metaChunk: true,
    lastUpdated: true,
    markdown: {
        config: (md) => {
            // VitePress does not apply `base` to plain links that point at static files in docs/public
            const defaultLinkOpen = md.renderer.rules.link_open ?? ((tokens, idx, options, env, self) => self.renderToken(tokens, idx, options));

            md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
                const href = tokens[idx].attrGet('href');

                if (href && /^\/(images|configurations)\//.test(href)) {
                    tokens[idx].attrSet('href', websiteBasePath + href.substring(1));
                }

                return defaultLinkOpen(tokens, idx, options, env, self);
            };
        }
    },
    sitemap: {
        hostname: `${websiteUrlRootUrl}/`,
    },
    themeConfig: {
        // https://vitepress.dev/reference/default-theme-config
        siteTitle: websiteName,
        logo: '/images/logo.png',
        socialLinks: [
            { icon: 'github', link: repositoryUrl }
        ],
        footer: {
            message: `Released under the <a href="${repositoryUrl}/blob/main/LICENSE">MIT License</a>. A fork of <a href="https://github.com/mayswind/ezbookkeeping">mayswind/ezbookkeeping</a>.`,
            copyright: 'Copyright © 2020-2026 MaysWind and contributors.'
        },
        editLink: {
            pattern: `${repositoryUrl}/edit/main/website/docs/:path`
        },
        search: {
            provider: 'local',
            options: {
                locales: {
                    zh_Hans: {
                        translations: {
                            button: {
                                buttonText: '搜索文档',
                                buttonAriaLabel: '搜索文档'
                            },
                            modal: {
                                displayDetails: '显示详情',
                                resetButtonTitle: '清除查询条件',
                                backButtonTitle: '返回',
                                noResultsText: '无法找到相关结果',
                                footer: {
                                    selectText: '选择',
                                    navigateText: '切换',
                                    closeText: '关闭'
                                }
                            }
                        }
                    }
                }
            }
        }
    },
    locales: {
        root: {
            lang: 'en',
            label: 'English',
            themeConfig: {
                nav: [
                    { text: 'Installation', link: '/installation/' },
                    { text: 'Configuration', link: '/configuration/' },
                    { text: 'Usage', link: '/basic_usage/' },
                    { text: 'API', link: '/httpapi/' },
                    { text: 'MCP', link: '/mcp/' },
                    {
                        text: 'Download',
                        items: [
                            { text: 'Release Notes', link: `${repositoryUrl}/releases` },
                            { text: 'Packaged Files', link: `${repositoryUrl}/releases` },
                            { text: 'Docker Images', link: 'https://hub.docker.com/r/krazyjakee/ezbookkeeping2' }
                        ]
                    },
                    { text: 'FAQ', link: '/faq/' }
                ],
                sidebar: [
                    {
                        text: 'Installation',
                        collapsed: false,
                        items: [
                            { text: 'Get Started', link: '/installation/' },
                            { text: 'System Requirements', link: '/installation/system_requirements' },
                            { text: 'Run with Docker', link: '/installation/installation-docker' },
                            { text: 'Run on Kubernetes', link: '/installation/installation-kubernetes' },
                            { text: 'Install from binary', link: '/installation/installation-binary' },
                            { text: 'Install with 1Panel (Community Supported)', link: '/installation/installation-1panel' },
                            { text: 'Reverse proxy configuration', link: '/installation/reverse-proxy-configuration' },
                            { text: 'Migrating from ezBookkeeping', link: '/installation/migrating' }
                        ]
                    },
                    {
                        text: 'Administration',
                        collapsed: false,
                        items: [
                            { text: 'Configuration', link: '/configuration/' },
                            { text: 'Exchange Rates', link: '/exchange_rates/' },
                            { text: 'Command Line', link: '/command_line/' },
                            { text: 'Troubleshooting', link: '/troubleshooting/'}
                        ]
                    },
                    {
                        text: 'Usage',
                        collapsed: false,
                        items: [
                            { text: 'Basic Usage', link: '/basic_usage/' },
                            { text: 'Insights Explorer', link: '/insights_explorer/' },
                            { text: 'Export & Import', link: '/export_and_import/' },
                            { text: 'Model Context Protocol', link: '/mcp/' },
                            { text: 'Agent Skill', link: '/agent/skill' },
                            { text: 'Integration with OpenClaw', link: '/agent/openclaw' }
                        ]
                    },
                    {
                        text: 'API',
                        collapsed: false,
                        items: [
                            { text: 'Introduction', link: '/httpapi/' },
                            { text: 'Token API', link: '/httpapi/token_api' },
                            { text: 'Account API', link: '/httpapi/account_api' },
                            { text: 'Transaction Category API', link: '/httpapi/transaction_category_api' },
                            { text: 'Transaction Tag API', link: '/httpapi/transaction_tag_api' },
                            { text: 'Transaction API', link: '/httpapi/transaction_api' },
                            { text: 'Exchange Rates API', link: '/httpapi/exchange_rate_api' }
                        ]
                    },
                    {
                        text: 'Contributing',
                        collapsed: false,
                        items: [
                            { text: 'Developing', link: '/developing/' },
                            { text: 'Translating', link: '/translating/' }
                        ]
                    },
                    {
                        text: 'Help',
                        collapsed: false,
                        items: [
                            { text: 'Feature List', link: '/features/' },
                            { text: 'FAQ', link: '/faq/' }
                        ]
                    }
                ],
                outline: {
                    level: [2, 3]
                }
            }
        },
        zh_Hans: {
            lang: 'zh-Hans',
            label: '中文 (简体)',
            link: '/zh_Hans',
            title: titleZhHans,
            description: descriptionZhHans,
            themeConfig: {
                nav: [
                    { text: '安装', link: '/zh_Hans/installation/' },
                    { text: '配置', link: '/zh_Hans/configuration/' },
                    { text: '使用', link: '/zh_Hans/basic_usage/' },
                    { text: 'API', link: '/zh_Hans/httpapi/' },
                    { text: 'MCP', link: '/zh_Hans/mcp/' },
                    {
                        text: '下载',
                        items: [
                            { text: '更新日志', link: `${repositoryUrl}/releases` },
                            { text: '打包文件', link: `${repositoryUrl}/releases` },
                            { text: 'Docker 镜像', link: 'https://hub.docker.com/r/krazyjakee/ezbookkeeping2' }
                        ]
                    },
                    { text: '常见问题', link: '/zh_Hans/faq/' }
                ],
                sidebar: [
                    {
                        text: '安装',
                        collapsed: false,
                        items: [
                            { text: '快速开始', link: '/zh_Hans/installation/' },
                            { text: '系统要求', link: '/zh_Hans/installation/system_requirements' },
                            { text: '通过 Docker 运行', link: '/zh_Hans/installation/installation-docker' },
                            { text: '在 Kubernetes 上运行', link: '/zh_Hans/installation/installation-kubernetes' },
                            { text: '从二进制包安装', link: '/zh_Hans/installation/installation-binary' },
                            { text: '使用 1Panel 安装（社区支持）', link: '/zh_Hans/installation/installation-1panel' },
                            { text: '反向代理的配置', link: '/zh_Hans/installation/reverse-proxy-configuration' }
                        ]
                    },
                    {
                        text: '管理',
                        collapsed: false,
                        items: [
                            { text: '配置', link: '/zh_Hans/configuration/' },
                            { text: '汇率', link: '/zh_Hans/exchange_rates/' },
                            { text: '命令行', link: '/zh_Hans/command_line/' },
                            { text: '问题排查', link: '/zh_Hans/troubleshooting/'}
                        ]
                    },
                    {
                        text: '使用',
                        collapsed: false,
                        items: [
                            { text: '基础使用', link: '/zh_Hans/basic_usage/' },
                            { text: '洞察探索', link: '/zh_Hans/insights_explorer/' },
                            { text: '导出 & 导入', link: '/zh_Hans/export_and_import/' },
                            { text: 'Model Context Protocol', link: '/zh_Hans/mcp/' },
                            { text: 'Agent 技能', link: '/zh_Hans/agent/skill' },
                            { text: '与 OpenClaw 集成', link: '/zh_Hans/agent/openclaw' }
                        ]
                    },
                    {
                        text: 'API',
                        collapsed: false,
                        items: [
                            { text: '介绍', link: '/zh_Hans/httpapi/' },
                            { text: '令牌 API', link: '/zh_Hans/httpapi/token_api' },
                            { text: '账户 API', link: '/zh_Hans/httpapi/account_api' },
                            { text: '交易分类 API', link: '/zh_Hans/httpapi/transaction_category_api' },
                            { text: '交易标签 API', link: '/zh_Hans/httpapi/transaction_tag_api' },
                            { text: '交易 API', link: '/zh_Hans/httpapi/transaction_api' },
                            { text: '汇率 API', link: '/zh_Hans/httpapi/exchange_rate_api' }
                        ]
                    },
                    {
                        text: '贡献',
                        collapsed: false,
                        items: [
                            { text: '开发', link: '/zh_Hans/developing/' },
                            { text: '翻译', link: '/zh_Hans/translating/' }
                        ]
                    },
                    {
                        text: '帮助',
                        collapsed: false,
                        items: [
                            { text: '功能列表', link: '/zh_Hans/features/' },
                            { text: '常见问题', link: '/zh_Hans/faq/' }
                        ]
                    }
                ],
                outline: {
                    label: '页面导航',
                    level: [2, 3]
                },
                editLink: {
                    text: '在 GitHub 上编辑此页面'
                },
                lastUpdated: {
                    text: '最后更新于'
                },
                docFooter: {
                    prev: '上一页',
                    next: '下一页'
                },
                darkModeSwitchLabel: '主题',
                lightModeSwitchTitle: '切换到浅色模式',
                darkModeSwitchTitle: '切换到深色模式',
                sidebarMenuLabel: '菜单',
                returnToTopLabel: '返回顶部',
                langMenuLabel: '选择语言'
            }
        }
    },
    transformPageData: (pageData, ctx) => {
        const site = resolveSiteDataByRoute(
            ctx.siteConfig.site,
            pageData.relativePath
        );

        const title: string = pageData.title ? `${pageData.title} | ${site.title}` : `${site.title}`;
        const description: string = `${pageData.description || site.description}`;
        const relativePath: string = pageData.relativePath
            .replace(/index\.md$/, '')
            .replace(/\.md$/, '');
        const pureDocumentRelativePath: string = relativePath
            .replace(/^zh_Hans\//i, '');

        const currentUrl: string = `${websiteUrlRootUrl}/${relativePath}`;
        const alternateEnglishUrl: string = `${websiteUrlRootUrl}/${pureDocumentRelativePath}`;
        const alternateChineseSimplifiedUrl: string = `${websiteUrlRootUrl}/zh_Hans/${pureDocumentRelativePath}`;

        const image: string = websiteUrlRootUrl + '/images/og-image.png';

        ((pageData.frontmatter.head ??= []) as HeadConfig[]).push(
            ['link', { rel: 'canonical', href: currentUrl }],
            ['link', { rel: 'alternate', hreflang: 'x-default', href: alternateEnglishUrl }],
            ['link', { rel: 'alternate', hreflang: 'en', href: alternateEnglishUrl }],
            ['link', { rel: 'alternate', hreflang: 'zh-Hans', href: alternateChineseSimplifiedUrl }],
            ['meta', { property: 'og:url', content: currentUrl }],
            ['meta', { property: 'og:title', content: title }],
            ['meta', { property: 'og:description', content: description }],
            ['meta', { property: 'og:image', content: image }],
            ['meta', { property: 'og:type', content: 'website' }],
            ['meta', { property: 'og:locale', content: site.lang }],
            ['meta', { property: 'og:site_name', content: websiteName }],
            ['meta', { property: 'twitter:card', content: 'summary_large_image' }],
            ['meta', { property: 'twitter:title', content: title }],
            ['meta', { property: 'twitter:site_name', content: description }],
            ['meta', { property: 'twitter:image', content: image }],
            [
                'script',
                { type: 'application/ld+json' },
                `{ "@context": "https://schema.org", "@type": "WebSite", "name": "${websiteName}", "headline": "${title}", "description": "${description}", "url": "${currentUrl}" }`
            ]
        );
    }
});
