import { computed, type Ref } from 'vue';

import type { StreamEvent } from '../api/client';

export interface ProgressStep {
  label: string;
  signals: Array<
    { type: 'phase'; value: string } | { type: 'status'; value: string }
  >;
}

export function resolveProgressIndex(
  events: StreamEvent[],
  steps: ProgressStep[],
): number {
  let index = 0;
  for (const event of events) {
    for (const [stepIndex, step] of steps.entries()) {
      if (step.signals.some((signal) => matchesSignal(event, signal))) {
        index = Math.max(index, stepIndex);
      }
    }
  }
  return index;
}

export function useProgressSteps(
  active: Ref<boolean>,
  steps: Ref<ProgressStep[]>,
  events: Ref<StreamEvent[]>,
) {
  const activeIndex = computed(() => {
    if (!active.value || steps.value.length === 0) {
      return 0;
    }
    return resolveProgressIndex(events.value, steps.value);
  });

  return { activeIndex };
}

function matchesSignal(
  event: StreamEvent,
  signal: { type: 'phase' | 'status'; value: string },
): boolean {
  if (signal.type === 'phase') {
    return event.type === 'phase' && event.phase === signal.value;
  }
  return event.type === 'status' && event.name === signal.value;
}
