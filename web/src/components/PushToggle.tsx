import { pushButtonLabel, type PushToggleState } from '../pushDevice';
import { usePushToggle } from '../usePushToggle';

export function PushToggleButton({ state, onClick }: { state: PushToggleState; onClick: () => void }) {
  if (state === 'loading' || state === 'unsupported') return null;
  return (
    <button
      type="button"
      className={`wall-push${state === 'on' ? ' on' : ''}`}
      aria-pressed={state === 'on'}
      disabled={state === 'denied'}
      onClick={onClick}
    >
      {pushButtonLabel(state)}
    </button>
  );
}

export function PushToggle() {
  const { state, toggle } = usePushToggle();
  return <PushToggleButton state={state} onClick={() => void toggle()} />;
}
