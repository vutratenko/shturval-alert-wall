import { createRoot } from 'react-dom/client';
import { App } from './App';
import { registerWallServiceWorker } from './pwa';
import './theme.css';

registerWallServiceWorker();
createRoot(document.getElementById('root')!).render(<App />);