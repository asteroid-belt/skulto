import { useEffect, useMemo, useState } from 'react';
import { Routes, Route } from 'react-router-dom';
import { loadConfig } from './config.js';
import Sidebar from './components/Sidebar.jsx';
import Landing from './components/Landing.jsx';
import PlanView from './components/PlanView.jsx';

// Below this width the sidebar becomes an overlay drawer and auto-collapses.
const NARROW = '(max-width: 860px)';
const isNarrow = () =>
  typeof window !== 'undefined' && window.matchMedia(NARROW).matches;

export default function App() {
  const [config, setConfig] = useState(null);
  const [query, setQuery] = useState('');
  // Open on wide screens, collapsed on narrow ones (resolved again on mount).
  const [navOpen, setNavOpen] = useState(() => !isNarrow());

  useEffect(() => {
    let alive = true;
    loadConfig().then((cfg) => {
      if (!alive) return;
      setConfig(cfg);
      if (cfg.title) document.title = cfg.title;
    });
    return () => {
      alive = false;
    };
  }, []);

  // Auto-collapse when the viewport is narrow; auto-open when it widens again.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia(NARROW);
    const apply = () => setNavOpen(!mq.matches);
    apply();
    mq.addEventListener('change', apply);
    return () => mq.removeEventListener('change', apply);
  }, []);

  const filtered = useMemo(() => {
    const cfg = config || { routes: [], plans: [] };
    const q = query.trim().toLowerCase();
    const match = (item) => {
      if (!q) return true;
      const haystack = [item.title, item.summary, ...(item.tags || [])]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return haystack.includes(q);
    };
    return {
      routes: (cfg.routes || []).filter(match),
      plans: (cfg.plans || []).filter(match),
    };
  }, [config, query]);

  if (!config) {
    return (
      <div className="app-loading">
        <div className="spinner" aria-hidden="true" />
        <p>Loading hub…</p>
      </div>
    );
  }

  // On narrow screens, picking an item should close the overlay drawer.
  const closeIfNarrow = () => {
    if (isNarrow()) setNavOpen(false);
  };

  return (
    <div className={'app-shell' + (navOpen ? '' : ' nav-collapsed')}>
      <Sidebar
        config={config}
        filtered={filtered}
        query={query}
        onQueryChange={setQuery}
        onToggle={() => setNavOpen((v) => !v)}
        onNavigate={closeIfNarrow}
      />
      <button
        type="button"
        className="nav-open-btn"
        onClick={() => setNavOpen(true)}
        aria-label="Open navigation"
        title="Open navigation"
      >
        ☰
      </button>
      <div
        className="nav-scrim"
        onClick={() => setNavOpen(false)}
        aria-hidden="true"
      />
      <main className="content">
        <Routes>
          <Route
            path="/"
            element={<Landing config={config} routes={filtered.routes} plans={filtered.plans} />}
          />
          <Route path="/plans/:name" element={<PlanView config={config} />} />
        </Routes>
      </main>
    </div>
  );
}
