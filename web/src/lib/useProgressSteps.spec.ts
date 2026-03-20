import { ref } from 'vue';
import { describe, expect, it } from 'vitest';

import type { StreamEvent } from '../api/client';
import {
  resolveProgressIndex,
  useProgressSteps,
  type ProgressStep,
} from './useProgressSteps';

describe('useProgressSteps', () => {
  const steps: ProgressStep[] = [
    {
      label: '保存回答',
      signals: [{ type: 'status', value: 'answer_received' }],
    },
    {
      label: '开始评估',
      signals: [{ type: 'status', value: 'evaluation_started' }],
    },
    {
      label: '反馈完成',
      signals: [{ type: 'status', value: 'feedback_ready' }],
    },
  ];

  it('resolves progress index from matching status events', () => {
    const events: StreamEvent[] = [
      { type: 'status', name: 'answer_received' },
      { type: 'status', name: 'evaluation_started' },
    ];

    expect(resolveProgressIndex(events, steps)).toBe(1);
  });

  it('supports phase-driven progress for create session flow', () => {
    const active = ref(true);
    const events = ref<StreamEvent[]>([
      { type: 'phase', phase: 'prepare_context' },
      { type: 'phase', phase: 'call_model' },
    ]);
    const createSteps = ref<ProgressStep[]>([
      {
        label: '读取上下文',
        signals: [{ type: 'phase', value: 'prepare_context' }],
      },
      {
        label: '调用模型',
        signals: [{ type: 'phase', value: 'call_model' }],
      },
      {
        label: '整理结果',
        signals: [{ type: 'phase', value: 'parse_result' }],
      },
    ]);

    const { activeIndex } = useProgressSteps(active, createSteps, events);
    expect(activeIndex.value).toBe(1);
  });
});
