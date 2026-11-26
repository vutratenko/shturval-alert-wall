import { takeSSEEvents, type SSEEvent } from './snapshot';

// fetch keeps the HTTP status visible. EventSource would hide a 401 from the wall.

export async function consumeSSE(response: Response, onEvent: (event: SSEEvent) => void): Promise<void> {
  if (!response.body) return;
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) return;
      buffer += decoder.decode(value, { stream: true });
      const parsed = takeSSEEvents(buffer);
      buffer = parsed.rest;
      for (const event of parsed.events) onEvent(event);
    }
  } finally {
    reader.releaseLock();
  }
}
