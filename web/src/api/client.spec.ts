import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  ApiError,
  createSessionStream,
  deleteSessions,
  downloadSessionBatchExport,
  getDashboard,
  submitAnswerStream,
} from './client';

function jsonResponse(body: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    status: init?.status,
  });
}

function ndjsonResponse(events: unknown[], init?: ResponseInit): Response {
  const body = `${events.map((event) => JSON.stringify(event)).join('\n')}\n`;
  return new Response(body, {
    headers: {
      'Content-Type': 'application/x-ndjson',
      ...(init?.headers ?? {}),
    },
    status: init?.status,
  });
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('api client', () => {
  it('surfaces structured API errors with code and status', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          {
            error: {
              code: 'job_target_not_ready',
              message: 'JD 还没准备好',
            },
          },
          { status: 409 },
        ),
      ),
    );

    await expect(getDashboard()).rejects.toMatchObject({
      name: 'ApiError',
      code: 'job_target_not_ready',
      status: 409,
      message: 'JD 还没准备好',
    });
  });

  it('parses NDJSON status events and returns the final session result', async () => {
    const onEvent = vi.fn();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        ndjsonResponse([
          { type: 'status', name: 'answer_received' },
          {
            type: 'trace',
            data: {
              flow: 'generate_question',
              phase: 'prepare_context',
              status: 'info',
              code: 'runtime_started',
              message: '开始执行 agent runtime。',
            },
          },
          {
            type: 'result',
            data: {
              id: 'sess_stream_1',
              mode: 'basics',
              topic: 'go',
              intensity: 'standard',
              status: 'waiting_answer',
              max_turns: 2,
              total_score: 0,
              turns: [],
            },
          },
        ]),
      ),
    );

    const session = await createSessionStream(
      {
        mode: 'basics',
        topic: 'go',
        intensity: 'standard',
        max_turns: 2,
      },
      onEvent,
    );

    expect(session.id).toBe('sess_stream_1');
    expect(onEvent).toHaveBeenCalledTimes(3);
    expect(onEvent).toHaveBeenNthCalledWith(1, {
      type: 'status',
      name: 'answer_received',
    });
    expect(onEvent).toHaveBeenNthCalledWith(2, {
      type: 'trace',
      data: {
        flow: 'generate_question',
        phase: 'prepare_context',
        status: 'info',
        code: 'runtime_started',
        message: '开始执行 agent runtime。',
      },
    });
  });

  it('throws ApiError when the stream emits an error event', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        ndjsonResponse([
          {
            type: 'error',
            code: 'session_busy',
            message: '当前会话正在处理中',
          },
        ]),
      ),
    );

    await expect(
      submitAnswerStream('sess_busy', '补一个案例', vi.fn()),
    ).rejects.toMatchObject({
      code: 'session_busy',
      message: '当前会话正在处理中',
    });
  });

  it('posts batch export requests with zip headers and resolves filenames', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response('zip-bytes', {
        status: 200,
        headers: {
          'Content-Disposition':
            'attachment; filename="practicehelper-sessions-2-json.zip"',
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const result = await downloadSessionBatchExport(
      ['sess_1', 'sess_2'],
      'json',
    );

    expect(result.filename).toBe('practicehelper-sessions-2-json.zip');
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/export', {
      method: 'POST',
      headers: {
        Accept: 'application/zip',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        session_ids: ['sess_1', 'sess_2'],
        format: 'json',
      }),
    });
  });

  it('posts delete requests with selected session ids', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        data: {
          deleted_count: 2,
          deleted_session_ids: ['sess_1', 'sess_2'],
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const result = await deleteSessions(['sess_1', 'sess_2']);

    expect(result.deleted_count).toBe(2);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/delete', {
      headers: {
        'Content-Type': 'application/json',
      },
      method: 'POST',
      body: JSON.stringify({
        session_ids: ['sess_1', 'sess_2'],
      }),
      signal: expect.any(AbortSignal),
    });
  });

  it('maps non-timeout aborts to canceled instead of timeout', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() =>
        Promise.reject(new DOMException('The user aborted a request.', 'AbortError')),
      ),
    );

    await expect(getDashboard()).rejects.toMatchObject<ApiError>({
      code: 'canceled',
      message: 'Request canceled',
    });
  });

  it('maps internal request timers to timeout', async () => {
    vi.useFakeTimers();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((_, init?: RequestInit) => {
        const signal = init?.signal;
        return new Promise((_, reject) => {
          signal?.addEventListener(
            'abort',
            () => reject(new DOMException('The operation was aborted.', 'AbortError')),
            { once: true },
          );
        });
      }),
    );

    const pending = expect(getDashboard()).rejects.toMatchObject<ApiError>({
      code: 'timeout',
      message: 'Request timeout',
    });
    await vi.advanceTimersByTimeAsync(90_000);

    await pending;
  });
});
