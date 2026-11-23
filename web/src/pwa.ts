const updateEveryMs = 60 * 60 * 1000;

export function registerWallServiceWorker(): void {
  if (!('serviceWorker' in navigator)) return;
  let hadController = !!navigator.serviceWorker.controller;
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (!hadController) {
      hadController = true;
      return;
    }
    window.location.reload();
  });
  void navigator.serviceWorker.register('/sw.js', { updateViaCache: 'none' }).then((reg) => {
    window.setInterval(() => {
      void reg.update();
    }, updateEveryMs);
  });
}
