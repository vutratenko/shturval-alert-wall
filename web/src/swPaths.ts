export function leaveToNetwork(pathname: string): boolean {
  return (
    pathname.startsWith('/api/') ||
    pathname.startsWith('/oauth2/') ||
    pathname === '/metrics' ||
    pathname.startsWith('/metrics/')
  );
}

export function isShellPath(pathname: string): boolean {
  return pathname === '/' || pathname === '/wall' || pathname === '/wall/';
}
