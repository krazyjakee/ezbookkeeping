<template>
    <p class="home-hero-eyebrow ebk-technical-label" v-if="frontmatter['hero'] && frontmatter['hero']['eyebrow']">
        {{ frontmatter['hero']['eyebrow'] }}
    </p>
</template>

<script setup>
import { useData } from 'vitepress';

const { frontmatter } = useData();
</script>

<style>
.home-hero-eyebrow {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 16px;
    color: var(--ebk-color-primary-text);
    line-height: 1.25;
}

.home-hero-eyebrow::before {
    content: '';
    flex: none;
    width: 8px;
    height: 8px;
    border-block-start: 2px solid var(--ebk-color-copper-line);
    border-inline-start: 2px solid var(--ebk-color-copper-line);
}

@media (forced-colors: active) {
    .home-hero-eyebrow::before {
        display: none;
    }
}
</style>
