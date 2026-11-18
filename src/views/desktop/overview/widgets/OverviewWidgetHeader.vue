<template>
    <header class="ebk-panel-header-strip overview-widget__header">
        <h2 class="overview-widget__title" :id="id" :title="title">{{ title }}</h2>
        <span class="overview-widget__meta ebk-numeric" :title="meta" v-if="meta">{{ meta }}</span>
        <div class="ebk-panel-header-actions overview-widget__actions" v-if="$slots['default']">
            <slot />
        </div>
    </header>
</template>

<script setup lang="ts">
defineProps<{
    id?: string;
    title: string;
    meta?: string;
}>();
</script>
