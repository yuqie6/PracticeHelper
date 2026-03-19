<template>
  <div class="min-h-screen bg-[var(--neo-bg)] text-[var(--neo-black)]">
    <header class="border-b-2 border-black bg-[var(--neo-red)] md:border-b-4">
      <div class="neo-page flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p class="neo-kicker bg-[var(--neo-yellow)]">PracticeHelper v0</p>
          <h1 class="text-2xl font-black uppercase tracking-[0.08em] md:text-4xl">
            会追杀弱点的面试训练 Agent
          </h1>
        </div>
        <nav class="flex flex-wrap gap-3">
          <RouterLink v-for="item in navItems" :key="item.to" :to="item.to" class="neo-button-dark">
            {{ item.label }}
          </RouterLink>
        </nav>
      </div>
    </header>

    <main>
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router';

const navItems = [
  { label: '首页', to: '/' },
  { label: '画像', to: '/profile' },
  { label: '项目', to: '/projects' },
  { label: '训练', to: '/train' },
];
</script>
