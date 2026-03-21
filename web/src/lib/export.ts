export const SESSION_EXPORT_FORMATS = ['markdown', 'json', 'pdf'] as const;
export type SessionExportFormat = (typeof SESSION_EXPORT_FORMATS)[number];
export const SESSION_EXPORT_FORMAT: SessionExportFormat = 'markdown';
export const SESSION_BATCH_EXPORT_PATH = '/api/sessions/export';

export function buildSessionExportPath(
  sessionId: string,
  format = SESSION_EXPORT_FORMAT,
): string {
  const params = new URLSearchParams({ format });
  return `/api/sessions/${encodeURIComponent(sessionId)}/export?${params.toString()}`;
}

export function fallbackSessionExportFilename(sessionId: string): string {
  return `practicehelper-session-${sessionId}.${exportExtension(
    SESSION_EXPORT_FORMAT,
  )}`;
}

export function fallbackSessionExportFilenameByFormat(
  sessionId: string,
  format: SessionExportFormat,
): string {
  return `practicehelper-session-${sessionId}.${exportExtension(format)}`;
}

export function fallbackBatchExportFilename(
  count: number,
  format: SessionExportFormat,
): string {
  return `practicehelper-sessions-${Math.max(count, 1)}-${format}.zip`;
}

export function resolveDownloadFilename(
  sessionId: string,
  format: SessionExportFormat,
  contentDisposition: string | null,
): string {
  return (
    parseContentDispositionFilename(contentDisposition) ??
    fallbackSessionExportFilenameByFormat(sessionId, format)
  );
}

export function resolveBatchDownloadFilename(
  count: number,
  format: SessionExportFormat,
  contentDisposition: string | null,
): string {
  return (
    parseContentDispositionFilename(contentDisposition) ??
    fallbackBatchExportFilename(count, format)
  );
}

export function triggerFileDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function parseContentDispositionFilename(value: string | null): string | null {
  if (!value) {
    return null;
  }

  const encoded = value.match(/filename\*=UTF-8''([^;]+)/i)?.[1];
  if (encoded) {
    try {
      return decodeURIComponent(encoded);
    } catch {
      return encoded;
    }
  }

  const plain = value.match(/filename="?([^";]+)"?/i)?.[1];
  return plain ?? null;
}

function exportExtension(format: SessionExportFormat): string {
  switch (format) {
    case 'json':
      return 'json';
    case 'pdf':
      return 'pdf';
    default:
      return 'md';
  }
}
