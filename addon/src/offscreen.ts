import { fetchSnapshot, type FetchOutcome } from './poll';

chrome.runtime.onMessage.addListener((message: { type?: string; baseUrl?: string }, _sender, sendResponse) => {
  if (message?.type !== 'poll-snapshot' || typeof message.baseUrl !== 'string') return;
  void fetchSnapshot(message.baseUrl).then((outcome: FetchOutcome) => sendResponse(outcome));
  return true;
});
