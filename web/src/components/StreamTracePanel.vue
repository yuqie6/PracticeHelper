<template>
  <div class="neo-panel bg-white space-y-4">
    <div>
      <p class="neo-kicker bg-[var(--neo-blue)]">{{ kicker }}</p>
      <h3 class="text-xl font-black uppercase tracking-[0.06em]">{{ title }}</h3>
      <p class="mt-2 text-base font-semibold">{{ description }}</p>
    </div>

    <div v-if="reasoning.length" class="space-y-2">
      <p class="neo-subheading">{{ reasoningTitle }}</p>
      <ul class="space-y-2">
        <li
          v-for="(item, index) in reasoning"
          :key="`${index}-${item}`"
          class="border-2 border-black bg-[var(--neo-paper)] px-3 py-3 text-sm font-semibold md:border-4"
        >
          {{ item }}
        </li>
      </ul>
    </div>

    <div v-if="content" class="space-y-2">
      <p class="neo-subheading">{{ contentTitle }}</p>
      <pre class="border-2 border-black bg-[var(--neo-paper)] px-3 py-3 text-sm font-semibold whitespace-pre-wrap md:border-4">{{ content }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  kicker: string;
  title: string;
  description: string;
  reasoningTitle: string;
  contentTitle: string;
  reasoning: string[];
  content: string;
}>();
</script>
