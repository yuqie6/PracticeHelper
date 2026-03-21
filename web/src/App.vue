<template>
  <AppShell>
    <RouterView v-slot="{ Component, route }">
      <Transition name="route" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </Transition>
    </RouterView>
  </AppShell>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router';

import AppShell from './components/AppShell.vue';
</script>
