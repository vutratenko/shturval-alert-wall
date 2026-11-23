import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { PushToggleButton } from './PushToggle';

describe('PushToggleButton', () => {
  it('offers to enable push on this device', () => {
    const html = renderToStaticMarkup(<PushToggleButton state="off" onClick={() => undefined} />);
    expect(html).toContain('Пуш выкл');
    expect(html).toContain('aria-pressed="false"');
  });

  it('shows that this device is subscribed', () => {
    const html = renderToStaticMarkup(<PushToggleButton state="on" onClick={() => undefined} />);
    expect(html).toContain('Пуш вкл');
    expect(html).toContain('aria-pressed="true"');
    expect(html).toContain('wall-push on');
  });

  it('stays disabled when the browser blocked notifications', () => {
    const html = renderToStaticMarkup(<PushToggleButton state="denied" onClick={() => undefined} />);
    expect(html).toContain('disabled');
    expect(html).toContain('Пуш запрещён');
  });

  it('stays off the wall until this browser can subscribe', () => {
    expect(renderToStaticMarkup(<PushToggleButton state="loading" onClick={() => undefined} />)).toBe('');
    expect(renderToStaticMarkup(<PushToggleButton state="unsupported" onClick={() => undefined} />)).toBe('');
  });
});
