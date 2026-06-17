import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { planName } from '../config.js';

function ThemeToggle() {
  const [theme, setTheme] = useState(
    () => document.documentElement.getAttribute('data-theme') || 'light'
  );
  const toggle = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try {
      localStorage.setItem('wt-theme', next);
    } catch (e) {
      /* ignore */
    }
    setTheme(next);
  };
  return (
    <button
      type="button"
      className="icon-btn theme-toggle"
      onClick={toggle}
      aria-label="Toggle light or dark theme"
      title="Toggle light/dark"
    >
      {theme === 'dark' ? '☾' : '☀'}
    </button>
  );
}

export default function Sidebar({ config, filtered, query, onQueryChange, onToggle, onNavigate }) {
  const location = useLocation();

  return (
    <aside className="sidebar">
      <div className="sidebar-head">
        <Link to="/" className="brand" onClick={onNavigate}>
          <span className="brand-mark" aria-hidden="true">◆</span>
          <span className="brand-text">{config.title || 'Walkthroughs'}</span>
        </Link>
        <div className="sidebar-head-actions">
          <ThemeToggle />
          <button
            type="button"
            className="icon-btn nav-collapse-btn"
            onClick={onToggle}
            aria-label="Collapse navigation"
            title="Collapse sidebar"
          >
            «
          </button>
        </div>
      </div>

      <div className="search">
        <input
          type="search"
          placeholder="Search by title or tag…"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          aria-label="Search walkthroughs and plans"
        />
      </div>

      <nav className="nav">
        <NavSection title="Walkthroughs" count={filtered.routes.length}>
          {filtered.routes.map((route) => (
            <a
              key={route.slug || route.file}
              className="nav-item"
              href={`#walkthrough-${route.slug || route.file}`}
              onClick={(e) => {
                // If we're already on landing, smooth-scroll to the card.
                if (location.pathname === '/') {
                  e.preventDefault();
                  const el = document.getElementById(
                    `walkthrough-${route.slug || route.file}`
                  );
                  el?.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
                onNavigate?.();
              }}
            >
              {route.title || route.slug}
            </a>
          ))}
          {filtered.routes.length === 0 && <p className="nav-empty">No matches</p>}
        </NavSection>

        <NavSection title="Plans" count={filtered.plans.length}>
          {filtered.plans.map((plan) => {
            const name = planName(plan);
            return (
              <Link
                key={name}
                className={
                  'nav-item' +
                  (location.pathname === `/plans/${name}` ? ' active' : '')
                }
                to={`/plans/${name}`}
                onClick={onNavigate}
              >
                {plan.title || name}
              </Link>
            );
          })}
          {filtered.plans.length === 0 && <p className="nav-empty">No matches</p>}
        </NavSection>
      </nav>

      <div className="sidebar-foot">
        <span>Built with the walkthrough skill</span>
      </div>
    </aside>
  );
}

function NavSection({ title, count, children }) {
  return (
    <div className="nav-section">
      <h3 className="nav-heading">
        {title}
        <span className="nav-count">{count}</span>
      </h3>
      <div className="nav-list">{children}</div>
    </div>
  );
}
