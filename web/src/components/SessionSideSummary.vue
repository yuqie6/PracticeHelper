<template>
  <aside class="session-side neo-stagger-list">
    <section v-if="showProgressPanel" class="session-side-panel" aria-live="polite">
      <ProgressPanel
        :kicker="t('session.processingKicker')"
        :title="progressTitle"
        :description="progressDescription"
        :steps="progressSteps"
        :active-index="progressStepIndex"
      />

      <div v-if="streamSections.length" class="max-h-[50vh] overflow-y-auto">
        <StreamTracePanel
          :kicker="t('session.processingKicker')"
          :title="progressTitle"
          :description="progressDescription"
          :reasoning-title="t('session.reasoningTitle')"
          :content-title="t('session.contentTitle')"
          :sections="streamSections"
        />
      </div>
    </section>

    <section class="neo-panel session-side-panel">
      <p class="neo-kicker bg-[var(--neo-blue)]">
        {{ t('session.jobTargetTitle') }}
      </p>
      <h2 class="session-section-title">
        {{ jobTargetTitle }}
      </h2>
      <p class="neo-note">
        {{ jobTargetDescription }}
      </p>
      <p v-if="promptSetSummary" class="neo-note">
        {{ promptSetSummary }}
      </p>
      <p v-if="promptOverlaySummary" class="neo-note">
        {{ t('session.promptOverlaySummary', { summary: promptOverlaySummary }) }}
      </p>
    </section>

    <section class="neo-panel session-side-panel">
      <p class="neo-kicker bg-[var(--neo-green)]">
        {{ t('session.feedback') }}
      </p>
      <FeedbackPanel v-if="latestEvaluation" :evaluation="latestEvaluation" />
      <p v-else class="neo-note">{{ t('session.feedbackEmpty') }}</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { TrainingEvaluation } from '../api/client';
import FeedbackPanel from './FeedbackPanel.vue';
import ProgressPanel from './ProgressPanel.vue';
import StreamTracePanel from './StreamTracePanel.vue';
import type { StreamSection } from '../lib/streaming';

defineProps<{
  showProgressPanel: boolean;
  progressTitle: string;
  progressDescription: string;
  progressSteps: string[];
  progressStepIndex: number;
  streamSections: StreamSection[];
  jobTargetTitle: string;
  jobTargetDescription: string;
  promptSetSummary: string;
  promptOverlaySummary: string;
  latestEvaluation: TrainingEvaluation | null;
}>();

const { t } = useI18n();
</script>

<style scoped>
.session-side {
  display: grid;
  gap: 1rem;
}

.session-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.session-section-title {
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

@media (min-width: 1280px) {
  .session-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
