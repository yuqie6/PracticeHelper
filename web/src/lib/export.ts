export const SESSION_EXPORT_FORMAT = 'markdown';

export function buildSessionExportPath(
  sessionId: string,
  format = SESSION_EXPORT_FORMAT,
): string {
  const params = new URLSearchParams({ format });
  return `/api/sessions/${encodeURIComponent(sessionId)}/export?${params.toString()}`;
}

export function fallbackSessionExportFilename(sessionId: string): string {
  return `practicehelper-session-${sessionId}.md`;
}

export function resolveDownloadFilename(
  sessionId: string,
  contentDisposition: string | null,
): string {
  return (
    parseContentDispositionFilename(contentDisposition) ??
    fallbackSessionExportFilename(sessionId)
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
