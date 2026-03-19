import { describe, expect, it } from 'vitest';

import { appendStreamEvent, parseStreamPayload, type StreamSection } from './streaming';

describe('streaming helpers', () => {
  it('splits repeated prepare_context phases into separate sections', () => {
    let sections: StreamSection[] = [];

    sections = appendStreamEvent(sections, { type: 'phase', phase: 'prepare_context' });
    sections = appendStreamEvent(sections, { type: 'context', name: 'read_evaluation_context' });
    sections = appendStreamEvent(sections, { type: 'reasoning', text: '正在评估主回答' });
    sections = appendStreamEvent(sections, { type: 'content', text: '{"score": 72}' });
    sections = appendStreamEvent(sections, { type: 'phase', phase: 'prepare_context' });
    sections = appendStreamEvent(sections, { type: 'reasoning', text: '正在整理复盘' });

    expect(sections).toHaveLength(2);
    expect(sections[0].contexts).toEqual(['read_evaluation_context']);
    expect(sections[0].rawContent).toContain('"score": 72');
    expect(sections[1].reasoning).toEqual(['正在整理复盘']);
  });

  it('parses question payload from streamed JSON text', () => {
    const payload = parseStreamPayload(
      '{"question":"Redis 为什么快？","expected_points":["内存","事件循环"]}',
    );

    expect(payload).toEqual({
      kind: 'question',
      question: 'Redis 为什么快？',
      expectedPoints: ['内存', '事件循环'],
    });
  });

  it('parses evaluation payload from streamed JSON text', () => {
    const payload = parseStreamPayload(
      '{"score":88,"score_breakdown":{"覆盖度":40},"strengths":["结论清楚"],"gaps":["细节不足"],"followup_question":"为什么？","followup_expected_points":["细节"]}',
    );

    expect(payload).toEqual({
      kind: 'evaluation',
      score: 88,
      scoreBreakdown: { 覆盖度: 40 },
      strengths: ['结论清楚'],
      gaps: ['细节不足'],
      followupQuestion: '为什么？',
      followupExpectedPoints: ['细节'],
    });
  });

  it('parses review payload from wrapped JSON text', () => {
    const payload = parseStreamPayload(
      '这里是结果：{"overall":"整体还行","highlights":["有结构"],"gaps":["缺细节"],"suggested_topics":["Redis"],"next_training_focus":["补案例"],"score_breakdown":{"表达":70}}',
    );

    expect(payload).toEqual({
      kind: 'review',
      overall: '整体还行',
      highlights: ['有结构'],
      gaps: ['缺细节'],
      suggestedTopics: ['Redis'],
      nextTrainingFocus: ['补案例'],
      scoreBreakdown: { 表达: 70 },
    });
  });
});
