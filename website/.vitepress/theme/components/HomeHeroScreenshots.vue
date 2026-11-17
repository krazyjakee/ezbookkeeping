<template>
    <div class="home-hero-screenshots" v-if="frontmatter['screenshots'] && frontmatter['screenshots'].length">
        <figure class="home-hero-screenshot"
                :class="screenshot['kind'] === 'mobile' ? 'is-mobile' : 'is-desktop'"
                :key="screenshot['light']"
                v-for="screenshot in frontmatter['screenshots']">
            <figcaption class="home-hero-screenshot-header ebk-technical-label" v-if="screenshot['label']">
                {{ screenshot['label'] }}
            </figcaption>
            <a class="home-hero-screenshot-link light" target="_blank" rel="noopener"
               :href="withBase(screenshot['light'])"
               :aria-label="screenshot['alt']">
                <img :src="withBase(screenshot['light'])" :alt="screenshot['alt']"
                     :width="screenshot['kind'] === 'mobile' ? 585 : 1440"
                     :height="screenshot['kind'] === 'mobile' ? 1266 : 1220"
                     loading="lazy" decoding="async" />
            </a>
            <a class="home-hero-screenshot-link dark" target="_blank" rel="noopener"
               :href="withBase(screenshot['dark'])"
               :aria-label="screenshot['alt']">
                <img :src="withBase(screenshot['dark'])" :alt="screenshot['alt']"
                     :width="screenshot['kind'] === 'mobile' ? 585 : 1440"
                     :height="screenshot['kind'] === 'mobile' ? 1266 : 1220"
                     loading="lazy" decoding="async" />
            </a>
        </figure>
    </div>
</template>

<script setup>
import { useData, withBase } from 'vitepress';

const { frontmatter } = useData();
</script>

<style>
html:not(.dark) .home-hero-screenshots .home-hero-screenshot-link.dark,
html.dark .home-hero-screenshots .home-hero-screenshot-link.light {
    display: none;
}

.home-hero-screenshots {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
    align-items: start;
    margin-top: 48px;
}

@media (min-width: 768px) {
    .home-hero-screenshots {
        /* desktop (1440×1220) and mobile (585×1266) frames end up the same height */
        grid-template-columns: minmax(0, 2.55fr) minmax(0, 1fr);
    }
}

.home-hero-screenshot {
    position: relative;
    margin: 0;
    overflow: hidden;
    border: 1px solid var(--ebk-color-hairline);
    border-radius: var(--ebk-radius-panel);
    background-color: var(--ebk-color-surface);
    box-shadow: var(--ebk-shadow-sm);
}

@media (max-width: 767px) {
    .home-hero-screenshot.is-mobile {
        justify-self: center;
        width: min(100%, 320px);
    }
}

/* copper start tick (D3) */
.home-hero-screenshot::before {
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

.home-hero-screenshot-header {
    display: flex;
    align-items: center;
    min-height: var(--ebk-header-strip-height);
    padding-inline: 20px;
    border-block-end: 1px solid var(--ebk-color-hairline);
    color: var(--ebk-color-text-secondary);
    line-height: 1.25;
}

.home-hero-screenshot-link {
    display: block;
    border-radius: 0 0 var(--ebk-radius-panel) var(--ebk-radius-panel);
}

.home-hero-screenshot-link:focus-visible {
    outline-offset: -4px !important;
}

.home-hero-screenshot-link > img {
    display: block;
    width: 100%;
    height: auto;
}

.home-hero-screenshot {
    transition: border-color var(--ebk-motion-duration-sm) var(--ebk-motion-ease-standard);
}

.home-hero-screenshot:hover {
    border-color: var(--ebk-color-copper-line);
}

@media (forced-colors: active) {
    .home-hero-screenshot::before {
        display: none;
    }

    .home-hero-screenshot {
        box-shadow: none;
    }
}
</style>
