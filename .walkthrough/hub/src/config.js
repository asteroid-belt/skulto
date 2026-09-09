// Runtime loading of walkthrough.config.json + path helpers.
//
// The build step (build-hub.mjs / CI) copies walkthrough.config.json and the
// content/ tree into this app's public/ folder, so everything is fetched
// relative to BASE_URL at runtime — no rebuild needed when content changes.

export const BASE_URL = import.meta.env.BASE_URL;

const EMPTY_CONFIG = {
  title: 'Walkthroughs',
  description: '',
  // Theme is a runtime light/dark toggle (default light, persisted in localStorage);
  // this key is informational only.
  theme: 'auto',
  routes: [],
  plans: [],
};

/** Strip directory portion from a config `file` path, e.g. content/routes/x.html -> x.html */
export function basename(filePath) {
  return String(filePath || '').split('/').pop();
}

/** URL to a standalone route HTML inside the published site. */
export function routeUrl(route) {
  return `${BASE_URL}content/routes/${basename(route.file)}`;
}

/** URL to a plan markdown file inside the published site. */
export function planUrl(plan) {
  return `${BASE_URL}content/plans/${basename(plan.file)}`;
}

/** Stable, URL-safe identifier for a plan (used in #/plans/:name). */
export function planName(plan) {
  return basename(plan.file).replace(/\.md$/i, '');
}

export async function loadConfig() {
  try {
    const res = await fetch(`${BASE_URL}walkthrough.config.json`, { cache: 'no-cache' });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    return {
      ...EMPTY_CONFIG,
      ...data,
      routes: Array.isArray(data.routes) ? data.routes : [],
      plans: Array.isArray(data.plans) ? data.plans : [],
    };
  } catch (err) {
    console.error('Failed to load walkthrough.config.json:', err);
    return { ...EMPTY_CONFIG, _error: String(err) };
  }
}
