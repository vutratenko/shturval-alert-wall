const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1']);

export function normalizeOrigin(input: string): string | null {
  let url: URL;
  try {
    url = new URL(input.trim());
  } catch {
    return null;
  }
  if (url.username || url.password || !url.hostname) return null;
  const local = LOCAL_HOSTS.has(url.hostname);
  if (url.protocol === 'https:') {
    return url.origin;
  }
  if (url.protocol === 'http:' && local) return url.origin;
  return null;
}

export function originPattern(origin: string): string {
  return `${origin.replace(/\/$/, '')}/*`;
}

export function wallUrl(baseUrl: string): string {
  return `${baseUrl}/wall`;
}

export function loginUrl(baseUrl: string): string {
  return `${baseUrl}/oauth2/login?rd=${encodeURIComponent('/wall')}`;
}
