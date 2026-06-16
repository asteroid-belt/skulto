import { useEffect, useMemo, useState } from 'react';
import { Routes, Route } from 'react-router-dom';
import { loadConfig } from './config.js';
import Sidebar from './components/Sidebar.jsx';
import Landing from './components/Landing.jsx';
import PlanView from './components/PlanView.jsx';

export default function App() {
  const [config, setConfig] = useState(null);
  const [query, setQuery] = useState('');

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

  return (
    <div className="app-shell">
      <Sidebar
        config={config}
        filtered={filtered}
        query={query}
        onQueryChange={setQuery}
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
