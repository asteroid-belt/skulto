import { Link } from 'react-router-dom';
import { routeUrl, planName } from '../config.js';

export default function Landing({ config, routes, plans }) {
  return (
    <div className="landing">
      <header className="landing-hero">
        <h1>{config.title || 'Walkthroughs'}</h1>
        {config.description && <p className="lede">{config.description}</p>}
      </header>

      {config._error && (
        <div className="banner banner-error">
          Could not load <code>walkthrough.config.json</code>. {config._error}
        </div>
      )}

      <section className="section" aria-labelledby="walkthroughs-heading">
        <h2 id="walkthroughs-heading" className="section-title">
          Walkthroughs
        </h2>
        {routes.length === 0 ? (
          <p className="empty-state">No walkthroughs match your search yet.</p>
        ) : (
          <div className="card-grid">
            {routes.map((route) => (
              <RouteCard key={route.slug || route.file} route={route} />
            ))}
          </div>
        )}
      </section>

      {plans.length > 0 && (
        <section className="section" aria-labelledby="plans-heading">
          <h2 id="plans-heading" className="section-title">
            Plans &amp; Design Docs
          </h2>
          <div className="plan-list">
            {plans.map((plan) => {
              const name = planName(plan);
              return (
                <Link key={name} to={`/plans/${name}`} className="plan-row">
                  <span className="plan-icon" aria-hidden="true">📄</span>
                  <span className="plan-body">
                    <span className="plan-title">{plan.title || name}</span>
                    {plan.summary && <span className="plan-summary">{plan.summary}</span>}
                  </span>
                  <span className="plan-arrow" aria-hidden="true">→</span>
                </Link>
              );
            })}
          </div>
        </section>
      )}
    </div>
  );
}

function RouteCard({ route }) {
  const id = `walkthrough-${route.slug || route.file}`;
  return (
    <a
      id={id}
      className="card"
      href={routeUrl(route)}
    >
      <div className="card-top">
        {route.audience && <span className="badge">{route.audience}</span>}
        {route.source && <span className="card-source">{route.source}</span>}
      </div>
      <h3 className="card-title">{route.title || route.slug}</h3>
      {route.summary && <p className="card-summary">{route.summary}</p>}
      {Array.isArray(route.tags) && route.tags.length > 0 && (
        <div className="chips">
          {route.tags.map((tag) => (
            <span key={tag} className="chip">
              {tag}
            </span>
          ))}
        </div>
      )}
      <span className="card-cta">Open walkthrough →</span>
    </a>
  );
}
