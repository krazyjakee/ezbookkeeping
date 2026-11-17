import { defineComponent, h, onMounted, watch } from 'vue';
import type { Theme } from 'vitepress';
import { useData } from 'vitepress';
import DefaultTheme from 'vitepress/theme';

import './custom.css';

import HomeHeroEyebrow from './components/HomeHeroEyebrow.vue';
import HomeHeroScreenshots from './components/HomeHeroScreenshots.vue';
import HomeSitemap from './components/HomeSitemap.vue';

// browser chrome follows the page canvas (docs/design/README.md §6.6)
const canvasLight: string = '#f7f4ef';
const canvasDark: string = '#0b0a09';

const Layout = defineComponent({
    name: 'EbkDocsLayout',
    setup() {
        const { isDark } = useData();

        onMounted(() => {
            watch(isDark, (dark) => {
                document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark ? canvasDark : canvasLight);
            }, { immediate: true });
        });

        return () => h(DefaultTheme.Layout, null, {
            "home-hero-info-before": () => h(HomeHeroEyebrow),
            "home-hero-actions-after": () => h(HomeHeroScreenshots),
            "home-features-after": () => h(HomeSitemap)
        });
    }
});

export default {
    extends: DefaultTheme,
    Layout
} satisfies Theme;
