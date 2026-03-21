import { describe, expect, it } from 'vitest';

import {
  buildSessionExportPath,
  fallbackSessionExportFilename,
  resolveDownloadFilename,
} from './export';

describe('export helpers', () => {
  it('builds the markdown export path', () => {
    expect(buildSessionExportPath('sess_1')).toBe(
      '/api/sessions/sess_1/export?format=markdown',
    );
  });

  it('uses header filename when present', () => {
    expect(
      resolveDownloadFilename(
        'sess_1',
        'attachment; filename="practicehelper-session-sess_1.md"',
      ),
    ).toBe('practicehelper-session-sess_1.md');
  });

  it('falls back to the default filename when the header is missing', () => {
    expect(resolveDownloadFilename('sess_2', null)).toBe(
      fallbackSessionExportFilename('sess_2'),
    );
  });
});
