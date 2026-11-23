import { describe, expect, it } from 'vitest';
import { pushButtonLabel, urlBase64ToUint8Array } from './pushDevice';

describe('push device toggle', () => {
  it('names the states a person sees on this browser', () => {
    expect(pushButtonLabel('off')).toBe('Пуш выкл');
    expect(pushButtonLabel('on')).toBe('Пуш вкл');
    expect(pushButtonLabel('denied')).toBe('Пуш запрещён');
  });

  it('decodes a url-safe VAPID key into bytes', () => {
    expect(Array.from(urlBase64ToUint8Array('YQ'))).toEqual([97]);
    expect(Array.from(urlBase64ToUint8Array('-_8'))).toEqual([251, 255]);
  });
});
