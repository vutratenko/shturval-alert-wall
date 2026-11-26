import { describe, expect, it } from 'vitest';
import { loginUrl, normalizeOrigin, originPattern, wallUrl } from './origin';

describe('normalizeOrigin', () => {
  it('keeps an https origin and drops the wall path', () => {
    expect(normalizeOrigin('https://alerts.example.com/wall')).toBe('https://alerts.example.com');
    expect(normalizeOrigin(' https://alerts.example.com/wall/ ')).toBe('https://alerts.example.com');
  });

  it('allows http only for localhost and loopback', () => {
    expect(normalizeOrigin('http://localhost:8080/wall')).toBe('http://localhost:8080');
    expect(normalizeOrigin('http://127.0.0.1:8080')).toBe('http://127.0.0.1:8080');
    expect(normalizeOrigin('http://alerts.example.com')).toBeNull();
  });

  it('rejects empty, non-http, and credentialed urls', () => {
    expect(normalizeOrigin('')).toBeNull();
    expect(normalizeOrigin('not a url')).toBeNull();
    expect(normalizeOrigin('ftp://alerts.example.com')).toBeNull();
    expect(normalizeOrigin('https://user:pass@alerts.example.com')).toBeNull();
  });
});

describe('wall paths', () => {
  it('builds the wall and login urls from the origin', () => {
    expect(wallUrl('https://alerts.example.com')).toBe('https://alerts.example.com/wall');
    expect(loginUrl('https://alerts.example.com')).toBe(
      'https://alerts.example.com/oauth2/login?rd=%2Fwall',
    );
    expect(originPattern('https://alerts.example.com')).toBe('https://alerts.example.com/*');
  });
});
