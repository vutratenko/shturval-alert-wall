export type PushToggleState = 'loading' | 'unsupported' | 'denied' | 'off' | 'on';

export function pushButtonLabel(state: PushToggleState): string {
  switch (state) {
    case 'on':
      return 'Пуш вкл';
    case 'denied':
      return 'Пуш запрещён';
    default:
      return 'Пуш выкл';
  }
}

export function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(base64);
  const buffer = new ArrayBuffer(raw.length);
  const out = new Uint8Array(buffer);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}
