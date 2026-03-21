import { describeJobTargetStatus, isJobTargetReady } from './jobTargetStatus';
import { messages } from '../i18n/messages';

function createTranslator(locale: 'zh-CN' | 'en' = 'zh-CN') {
  return (key: string, named?: Record<string, unknown>) => {
    const value = key.split('.').reduce<unknown>((current, segment) => {
      if (!current || typeof current !== 'object') {
        return undefined;
      }

      return (current as Record<string, unknown>)[segment];
    }, messages[locale]);

    if (typeof value !== 'string') {
      return key;
    }

    return value.replace(/\{(\w+)\}/g, (_, token) =>
      String(named?.[token] ?? ''),
    );
  };
}

describe('job target status helpers', () => {
  const t = createTranslator();

  it('treats only succeeded status as ready for training', () => {
    expect(isJobTargetReady('succeeded')).toBe(true);
    expect(isJobTargetReady('stale')).toBe(false);
    expect(isJobTargetReady('failed')).toBe(false);
    expect(isJobTargetReady(undefined)).toBe(false);
  });

  it('returns status-specific copy for training fallback', () => {
    expect(
      describeJobTargetStatus(t, 'trainFallback', 'stale', {
        name: '后端工程师 JD',
      }),
    ).toContain('原文已变更');
    expect(
      describeJobTargetStatus(t, 'trainFallback', 'running', {
        name: '后端工程师 JD',
      }),
    ).toContain('正在分析中');
  });

  it('falls back to unknown copy for unsupported status', () => {
    expect(
      describeJobTargetStatus(t, 'jobsReadiness', 'unexpected-status' as never),
    ).toContain('暂时无法确认');
  });
});
