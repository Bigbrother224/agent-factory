#!/usr/bin/env node
/**
 * Dashboard de l'usine — agent-factory (v3 : atelier en direct)
 * Sert localhost:7777. GitHub reste l'unique bus :
 *  - LECTURE : issues, PRs, runs, jobs/étapes en cours, logs live,
 *    commentaires, verdicts kimi, roadmap des chantiers, portfolio.
 *  - ÉCRITURE : uniquement les décisions humaines (labels, merge, notes/!fix).
 */
const http = require('http');
const { execFile } = require('child_process');
const fs = require('fs');
const path = require('path');

const REPO = process.env.FACTORY_REPO || 'Bigbrother224/agent-factory';
const PORT = parseInt(process.env.PORT || '7777', 10);
const HTML = path.join(__dirname, 'dashboard.html');
const AXES = ['ai-automation-and-systems', 'embedded-intelligence', 'experimental-computing'];

function gh(args) {
  return new Promise((resolve) => {
    execFile('gh', args, { timeout: 25000, maxBuffer: 20 * 1024 * 1024 }, (err, stdout, stderr) => {
      resolve({ ok: !err, out: stdout, err: stderr || (err && err.message) || '' });
    });
  });
}
async function ghJson(args) {
  const r = await gh(args);
  if (!r.ok) return { ok: false, data: null, err: r.err };
  try { return { ok: true, data: JSON.parse(r.out), err: '' }; }
  catch (e) { return { ok: false, data: null, err: 'JSON invalide: ' + e.message }; }
}
const ghList = (args) => ghJson([...args, '-R', REPO]);
const ghApi = (rest) => ghJson(['api', `repos/${REPO}/${rest}`]);

/* Même slug que fleet/slugify.py : 3 mots, max 24 car. */
function slugify(title) {
  const s = String(title).normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-zA-Z0-9]+/g, '-').toLowerCase().replace(/^-+|-+$/g, '');
  const w = s.split('-').filter(Boolean);
  return w.slice(0, 3).join('-').slice(0, 24).replace(/-+$/g, '') || 'projet';
}

/* ROADMAP.md d'un chantier : tranches faites / total + tranche courante */
function parseRoadmap(md) {
  const lines = md.split(/\r?\n/).filter((l) => /- \[( |x|X)\] T\d/i.test(l));
  if (!lines.length) return null;
  const done = lines.filter((l) => /- \[[xX]\]/.test(l)).length;
  const nxt = lines.find((l) => /- \[ \]/.test(l));
  const current = nxt
    ? nxt.replace(/^- \[ \] T\d+\s*:?\s*/i, '').replace(/— fichiers:.*$/i, '').slice(0, 100).trim()
    : null;
  return { total: lines.length, done, current, finished: !nxt };
}

/* Revue kimi : structure complète — verdict, problèmes par gravité,
   notes qualité par dimension, bilan. Plus d'extrait tronqué :
   l'humain doit pouvoir TOUT lire depuis le dashboard. */
function parseReview(body, verdict, comment) {
  const problems = [];
  const notes = [];
  let bilan = '';
  let section = '';
  for (const line of String(body).split(/\r?\n/)) {
    const h = line.match(/^#{2,3}\s+(.+?)\s*$/);
    if (h) {
      const t = h[1].toLowerCase();
      section = /probl/.test(t) ? 'problems'
        : /note/.test(t) ? 'notes'
        : /bilan/.test(t) ? 'bilan' : 'verdict';
      continue;
    }
    if (section === 'bilan') { bilan += (bilan ? ' ' : '') + line.trim(); continue; }
    const p = line.match(/^-\s*\[([^\]]+)\]\s*(.+)$/);
    if (p && section === 'problems') { problems.push({ sev: p[1].trim().toUpperCase(), text: p[2].trim() }); continue; }
    const n = line.match(/^-\s*(.+?)\s*:\s*(\d)\s*\/\s*5\s*[—–-]\s*(.+)$/);
    if (n && section === 'notes') { notes.push({ dim: n[1].trim(), score: parseInt(n[2], 10), text: n[3].trim() }); }
  }
  return {
    verdict, at: comment.createdAt, url: comment.url,
    problems, notes, bilan: bilan.trim(), raw: String(body),
  };
}

async function fetchState() {
  const state = { repo: REPO, when: new Date().toISOString(), errors: [], axes: AXES };
  const [issues, prs, runs] = await Promise.all([
    ghList(['issue', 'list', '--state', 'all', '--limit', '50', '--json',
      'number,title,state,labels,createdAt,updatedAt,url']),
    ghList(['pr', 'list', '--state', 'all', '--limit', '40', '--json',
      'number,title,state,labels,headRefName,statusCheckRollup,createdAt,updatedAt,url']),
    ghList(['run', 'list', '--limit', '30', '--json',
      'databaseId,displayTitle,workflowName,status,conclusion,createdAt,event,headBranch']),
  ]);
  if (issues.ok) state.issues = issues.data; else state.errors.push('issues: ' + issues.err);
  if (prs.ok) state.prs = prs.data; else state.errors.push('prs: ' + prs.err);
  if (runs.ok) state.runs = runs.data; else state.errors.push('runs: ' + runs.err);
  if (!state.issues || !state.prs) return state;

  const hasL = (o, n) => (o.labels || []).some((l) => l.name === n);
  const axeOf = (i) => { const l = (i.labels || []).find((x) => x.name.startsWith('axe/')); return l ? l.name.slice(4) : null; };

  /* --- Atelier en direct : jobs/étapes + logs des runs actifs --- */
  state.live = [];
  const active = (state.runs || []).filter((r) => r.status === 'in_progress' || r.status === 'queued').slice(0, 3);
  for (const r of active) {
    const jr = await ghApi(`actions/runs/${r.databaseId}/jobs`);
    if (!jr.ok || !jr.data || !jr.data.jobs || !jr.data.jobs.length) {
      state.live.push({ id: r.databaseId, workflow: r.workflowName, title: r.displayTitle, status: r.status, startedAt: r.createdAt, steps: [] });
      continue;
    }
    const job = jr.data.jobs.find((j) => j.status === 'in_progress') || jr.data.jobs[0];
    const entry = {
      id: r.databaseId, workflow: r.workflowName, title: r.displayTitle,
      status: r.status, startedAt: r.createdAt,
      steps: (job.steps || []).map((s) => ({
        name: s.name, status: s.status, conclusion: s.conclusion,
        startedAt: s.startedAt, completedAt: s.completedAt,
      })),
    };
    if (job.status === 'in_progress') {
      const log = await gh(['api', `repos/${REPO}/actions/jobs/${job.id}/logs`]);
      if (log.ok) entry.logTail = log.out.slice(-4000).split(/\r?\n/).slice(-40).join('\n');
    }
    state.live.push(entry);
  }

  /* --- Fil de communication : commentaires des issues ouvertes --- */
  state.comments = {};
  const openIssues = state.issues.filter((i) => i.state === 'OPEN' &&
    !(i.labels || []).some((l) => l.name === 'documentation')).slice(0, 10);
  const issueComments = await Promise.all(openIssues.map((i) =>
    ghList(['issue', 'view', String(i.number), '--json', 'comments'])));
  issueComments.forEach((r, k) => {
    if (!r.ok || !r.data) return;
    state.comments[openIssues[k].number] = (r.data.comments || []).slice(-5).map((c) => ({
      author: c.author ? c.author.login : '?', at: c.createdAt, body: c.body || '',
    }));
  });

  /* --- Revues kimi + commentaires des PRs ouvertes --- */
  state.reviews = {}; state.prComments = {}; state.ci = {};
  const openPrs = state.prs.filter((p) => p.state === 'OPEN').slice(0, 10);
  const prViews = await Promise.all(openPrs.map((p) =>
    ghList(['pr', 'view', String(p.number), '--json', 'comments'])));
  prViews.forEach((r, k) => {
    const pr = openPrs[k];
    const rollup = pr.statusCheckRollup || [];
    const failed = rollup.filter((c) => c.conclusion && c.conclusion !== 'success' && c.conclusion !== 'skipped');
    const pending = rollup.filter((c) => !c.conclusion);
    state.ci[pr.number] = rollup.length === 0 ? 'none'
      : failed.length ? 'red' : pending.length ? 'pending' : 'green';
    if (!r.ok || !r.data) return;
    const comments = r.data.comments || [];
    state.prComments[pr.number] = comments.slice(-4).map((c) => ({
      author: c.author ? c.author.login : '?', at: c.createdAt, body: c.body || '',
    }));
    for (let j = comments.length - 1; j >= 0; j--) {
      const c = comments[j];
      const b = c.body || '';
      if (b.startsWith('## Revue du groupe C')) {
        const m = b.match(/\*\*(GO|NO-GO|NO GO)\*\*/i);
        state.reviews[pr.number] = m
          ? parseReview(b, m[1].toUpperCase().replace('NO GO', 'NO-GO'), c)
          : { verdict: '?', at: c.createdAt, url: c.url, problems: [], notes: [], bilan: '', raw: b };
        break;
      }
    }
  });

  /* --- Roadmap des chantiers : avancement des tranches (branche du PR) --- */
  state.roadmap = {};
  const chantierPrs = openPrs.filter((p) => /^(chantier|fleet)\//.test(p.headRefName)).slice(0, 4);
  for (const p of chantierPrs) {
    const slug = p.headRefName.split('/').pop();
    const issue = state.issues.find((i) => {
      const s = slugify(i.title);
      const t = i.title.toLowerCase();
      return axeOf(i) && (s === slug || s.includes(slug) || t.includes(slug));
    });
    if (!issue) continue;
    const axe = axeOf(issue);
    const rr = await ghApi(`contents/projects/${axe}/${slug}/ROADMAP.md?ref=${encodeURIComponent(p.headRefName)}`);
    if (!rr.ok || !rr.data || !rr.data.content) continue;
    try {
      const md = Buffer.from(rr.data.content, 'base64').toString('utf8');
      const rm = parseRoadmap(md);
      if (rm) state.roadmap[p.number] = rm;
    } catch (e) { /* roadmap illisible : ignorée */ }
  }

  /* --- Dernier digest de veille : ideas/YYYY-MM-DD-*.md le plus récent --- */
  state.lastDigest = null;
  const dg = await ghApi('contents/ideas');
  if (dg.ok && Array.isArray(dg.data)) {
    const dates = dg.data.map((f) => ((f.name || '').match(/^(\d{4}-\d{2}-\d{2})-/) || [])[1]).filter(Boolean).sort();
    if (dates.length) state.lastDigest = dates[dates.length - 1];
  }

  /* --- Portfolio : contenu réel de projects/<axe>/ sur main --- */
  state.portfolio = {};
  await Promise.all(AXES.map(async (axe) => {
    const r = await ghApi(`contents/projects/${axe}`);
    if (!r.ok || !Array.isArray(r.data)) { state.portfolio[axe] = []; return; }
    const dirs = r.data.filter((e) => e.type === 'dir').slice(0, 5);
    const projects = await Promise.all(dirs.map(async (d) => {
      const rr = await ghApi(`contents/projects/${axe}/${d.name}/README.md`);
      return { slug: d.name, readme: rr.ok };
    }));
    state.portfolio[axe] = projects;
  }));

  return state;
}

/* ---------- Cache (budget API GitHub maîtrisé) ---------- */
let cache = { when: 0, state: null };
async function getState() {
  if (cache.state && Date.now() - cache.when < 20000) return cache.state;
  const s = await fetchState();
  if (s.issues && s.prs) cache = { when: Date.now(), state: s };
  return cache.state || s;
}

/* ---------- Actions : uniquement les décisions humaines ---------- */
const ACTIONS = {
  approve_idea: (n) => gh(['issue', 'edit', String(n), '-R', REPO, '--add-label', 'idea/approved']),
  reject_idea: async (n, reason) => {
    await gh(['issue', 'comment', String(n), '-R', REPO, '--body',
      '🚫 Rejetée par l\'humain. Raison : ' + String(reason || 'non précisée')]);
    await gh(['issue', 'edit', String(n), '-R', REPO, '--add-label', 'idea/rejected']);
  },
  set_axe: (n, axe) => {
    const a = AXES.includes(String(axe)) ? String(axe) : null;
    if (!a) return Promise.resolve({ ok: false, error: 'axe inconnu (doit être un des 3 axes)' });
    return gh(['issue', 'edit', String(n), '-R', REPO, '--add-label', 'axe/' + a]);
  },
  retry_fleet: async (n) => {
    await gh(['issue', 'edit', String(n), '-R', REPO, '--remove-label', 'idea/approved']);
    await gh(['issue', 'edit', String(n), '-R', REPO, '--add-label', 'idea/approved']);
  },
  close_issue: (n) => gh(['issue', 'close', String(n), '-R', REPO, '--reason', 'not planned']),
  merge_pr: (n) => gh(['pr', 'merge', String(n), '-R', REPO, '--merge', '--delete-branch']),
  close_pr: (n) => gh(['pr', 'close', String(n), '-R', REPO]),
  comment_pr: (n, body) => gh(['pr', 'comment', String(n), '-R', REPO, '--body', String(body || '').slice(0, 2000)]),
  rerun: (id) => gh(['run', 'rerun', String(id), '-R', REPO]),
  /* Point de communication humain ↔ agents : commentaire d'issue.
     mode 'fix' → préfixé « !fix » → workflow retour-humain → session Aider. */
  note: async (n, extra) => {
    const text = String((extra && extra.text) || '').trim();
    if (!text) return { ok: false, error: 'message vide' };
    const mode = extra.mode === 'fix' ? 'fix' : 'note';
    const body = mode === 'fix' ? '!fix ' + text : text;
    return gh(['issue', 'comment', String(n), '-R', REPO, '--body', body.slice(0, 4000)]);
  },
};

const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && (req.url === '/' || req.url.startsWith('/index'))) {
    res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' });
    res.end(fs.readFileSync(HTML));
  } else if (req.method === 'GET' && req.url.startsWith('/api/state')) {
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end(JSON.stringify(await getState()));
  } else if (req.method === 'POST' && req.url.startsWith('/api/action')) {
    let body = '';
    req.on('data', (c) => { body += c; if (body.length > 10000) req.destroy(); });
    req.on('end', async () => {
      try {
        const { type, target, extra } = JSON.parse(body || '{}');
        const n = parseInt(target, 10);
        if (isNaN(n)) throw new Error('cible invalide');
        if (!ACTIONS[type]) throw new Error('action inconnue: ' + type);
        const r = await ACTIONS[type](n, extra);
        cache = { when: 0, state: null };
        res.writeHead(r && r.ok === false ? 500 : 200, { 'content-type': 'application/json' });
        res.end(JSON.stringify(r || { ok: true }));
      } catch (e) {
        res.writeHead(400, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ ok: false, error: e.message }));
      }
    });
  } else { res.writeHead(404); res.end('not found'); }
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`🏭 Dashboard usine v3 : http://localhost:${PORT}  (repo ${REPO})`);
  console.log('   Atelier en direct : étapes + logs des runs actifs. Notes : Entrée = envoyer.');
});