<template>
    <nav class="home-footer-sitemap" :aria-label="frontmatter['sitemapLabel'] || 'Sitemap'"
         v-if="frontmatter['sitemap'] && frontmatter['sitemap'].length">
        <div class="home-footer-sitemap-container">
            <section class="home-footer-sitemap-col" :key="groupIndex"
                     v-for="(linkGroup, groupIndex) in frontmatter['sitemap']">
                <h2 class="home-footer-sitemap-title ebk-technical-label">{{ linkGroup['name'] }}</h2>
                <ul>
                    <li :key="linkIndex" v-for="(link, linkIndex) in linkGroup['links']">
                        <a class="VPLink link"
                           :class="{ 'vp-external-link-icon': !!link['external'] }"
                           :href="link['external'] ? link['url'] : withBase(link['url'])"
                           :target="link['external'] ? '_blank' : undefined"
                           :rel="link['external'] ? 'noopener' : undefined"
                        >{{ link['name'] }}</a>
                    </li>
                </ul>
            </section>
        </div>
    </nav>
</template>

<script setup>
import { useData, withBase } from 'vitepress';

const { frontmatter } = useData();
</script>

<style scoped>
.home-footer-sitemap {
    margin-top: 96px;
    padding: 40px 24px 24px;
    border-top: 1px solid var(--ebk-color-hairline);
    background-color: var(--ebk-docs-footer-background-color);
}

.home-footer-sitemap-container {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
    max-width: 1152px;
    margin: 0 auto;
}

@media (min-width: 640px) {
    .home-footer-sitemap {
        padding-inline: 48px;
    }

    .home-footer-sitemap-container {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}

@media (min-width: 960px) {
    .home-footer-sitemap {
        padding-inline: 64px;
    }

    .home-footer-sitemap-container {
        grid-template-columns: repeat(3, minmax(0, 1fr));
    }
}

.home-footer-sitemap-col {
    position: relative;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-panel);
    background-color: var(--ebk-color-surface);
}

/* copper start tick (D3) */
.home-footer-sitemap-col::before {
    content: '';
    position: absolute;
    inset-block-start: 6px;
    inset-inline-start: 6px;
    width: 8px;
    height: 8px;
    border-block-start: 2px solid var(--ebk-color-copper-line);
    border-inline-start: 2px solid var(--ebk-color-copper-line);
    pointer-events: none;
}

.home-footer-sitemap-title {
    display: flex;
    align-items: center;
    min-height: var(--ebk-header-strip-height);
    margin: 0;
    padding-inline: 20px;
    border-block-end: 1px solid var(--ebk-color-hairline);
    color: var(--ebk-color-text-secondary);
    line-height: 1.25;
}

ul {
    margin: 0;
    padding: 8px 0;
    list-style: none;
}

li > a {
    display: block;
    padding: 6px 20px;
    color: var(--ebk-color-text-primary);
    font-size: 0.9rem;
    line-height: 1.5;
    transition: color var(--ebk-motion-duration-xs) var(--ebk-motion-ease-standard);
}

li > a:hover {
    color: var(--ebk-color-primary-text);
}

li > a:focus-visible {
    outline-offset: -2px !important;
}

@media (forced-colors: active) {
    .home-footer-sitemap-col::before {
        display: none;
    }
}
</style>
