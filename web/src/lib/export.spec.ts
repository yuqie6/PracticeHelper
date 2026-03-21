import { describe, expect, it } from 'vitest';

import {
  fallbackBatchExportFilename,
  buildSessionExportPath,
  resolveBatchDownloadFilename,
  fallbackSessionExportFilename,
  fallbackSessionExportFilenameByFormat,
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
        'markdown',
        'attachment; filename="practicehelper-session-sess_1.md"',
      ),
    ).toBe('practicehelper-session-sess_1.md');
  });

  it('falls back to the default filename when the header is missing', () => {
    expect(resolveDownloadFilename('sess_2', 'markdown', null)).toBe(
      fallbackSessionExportFilename('sess_2'),
    );
  });

  it('falls back to the batch filename when the archive header is missing', () => {
    expect(resolveBatchDownloadFilename(3, 'json', null)).toBe(
      fallbackBatchExportFilename(3, 'json'),
    );
  });

  it('builds json and pdf fallback filenames with the right extension', () => {
    expect(fallbackSessionExportFilenameByFormat('sess_json', 'json')).toBe(
      'practicehelper-session-sess_json.json',
    );
    expect(fallbackSessionExportFilenameByFormat('sess_pdf', 'pdf')).toBe(
      'practicehelper-session-sess_pdf.pdf',
    );
  });
});
