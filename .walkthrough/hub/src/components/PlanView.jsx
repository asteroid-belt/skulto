import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { planUrl, planName } from '../config.js';

export default function PlanView({ config }) {
  const { name } = useParams();
  const [state, setState] = useState({ status: 'loading', text: '' });

  const plan = (config.plans || []).find((p) => planName(p) === name);

  useEffect(() => {
    if (!plan) {
      setState({ status: 'missing', text: '' });
      return;
    }
    let alive = true;
    setState({ status: 'loading', text: '' });
    fetch(planUrl(plan), { cache: 'no-cache' })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.text();
      })
      .then((text) => alive && setState({ status: 'ready', text }))
      .catch((err) => alive && setState({ status: 'error', text: String(err) }));
    return () => {
      alive = false;
    };
  }, [plan, name]);

  if (!plan || state.status === 'missing') {
    return (
      <div className="plan-view">
        <p className="empty-state">Plan “{name}” not found.</p>
        <Link to="/" className="back-link">← Back to hub</Link>
      </div>
    );
  }

  return (
    <article className="plan-view">
      <div className="plan-view-head">
        <Link to="/" className="back-link">← Back to hub</Link>
        <h1>{plan.title || name}</h1>
        {plan.summary && <p className="lede">{plan.summary}</p>}
      </div>

      {state.status === 'loading' && <p className="empty-state">Loading…</p>}
      {state.status === 'error' && (
        <div className="banner banner-error">Failed to load plan: {state.text}</div>
      )}
      {state.status === 'ready' && (
        <div className="markdown">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              a: ({ node, ...props }) => (
                <a {...props} target="_blank" rel="noopener noreferrer" />
              ),
            }}
          >
            {state.text}
          </ReactMarkdown>
        </div>
      )}
    </article>
  );
}
