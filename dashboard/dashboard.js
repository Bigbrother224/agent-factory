#!/usr/bin/env node
/**
 * Dashboard de l'usine — agent-factory (v2, structure portfolio 3 axes)
 * Sert localhost:7777. GitHub reste l'unique bus :
 *  - LECTURE : issues, PRs, runs, commentaires, verdicts kimi, contenu du portfolio.
 *  - ÉCRITURE : uniquement les décisions humaines (labels, merge, notes/!fix).
 * Le composeur de notes publie des commentaires d'issue :
 *  - mode "fix"  → « !fix … » → workflow retour-humain → la flotte exécute.
 *  - mode "note" → commentaire simple → visible par toi, le scout et toute relecture.
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
async function ghApi(rest) { return ghJson(['api', `repos/${REPO}/${rest}`]); }

/* ---------- Lecture de l'état complet ---------- */
async function fetchState() {
  const state = { repo: REPO, when: new Date().toISOString(), errors: [], axes: AXES };
  const [issues, prs, runs, veille] = await Promise.all([
    ghList(['issue', 'list', '--state', 'all', '--limit', '50', '--json',
      'number,title,state,labels,body,createdAt,updatedAt,url']),
    ghList(['pr', 'list', '--state', 'all', '--limit', '40', '--json',
      'number,title,state,labels,headRefName,baseRefName,statusCheckRollup,createdAt,updatedAt,url']),
    ghList(['run', 'list', '--limit', '30', '--json',
      'databaseId,displayTitle,workflowName,status,conclusion,createdAt,event,headBranch']),
    ghApi('contents/.veille'),
  ]);
  if (issues.ok) state.issues = issues.data; else state.errors.push('issues: ' + issues.err);
  if (prs.ok) state.prs = prs.data; else state.errors.push('prs: ' + prs.err);
  if (runs.ok) state.runs = runs.data; else state.errors.push('runs: ' + runs.err);
  if (veille.ok && Array.isArray(veille.data)) {
    const digests = veille.data.filter(f => /^[\d-]+\.md$/.test(f.name)).map(f => f.name.replace('.md', ''));
    state.lastDigest = digests.sort().pop() || null;
  }
  if (!state.issues || !state.prs) return state;

  /* Fil de communication : commentaires des issues ouvertes (les tiens + ceux de l'usine) */
  state.comments = {};
  const openIssues = state.issues.filter(i => i.state === 'OPEN' &&
    !(i.labels || []).some(l => l.name === 'documentation')).slice(0, 10);
  const issueComments = await Promise.all(openIssues.map(i =>
    ghList(['issue', 'view', String(i.number), '--json', 'comments'])));
  issueComments.forEach((r, k) => {
    if (!r.ok || !r.data) return;
    state.comments[openIssues[k].number] = (r.data.comments || []).slice(-5).map(c => ({
      author: c.author ? c.author.login : '?',
      at: c.createdAt, body: c.body || '',
    }));
  });

  /* Revues kimi + derniers commentaires des PRs ouvertes */
  state.reviews = {}; state.prComments = {}; state.ci = {};
  const openPrs = state.prs.filter(p => p.state === 'OPEN').slice(0, 10);
  const prViews = await Promise.all(openPrs.map(p =>
    ghList(['pr', 'view', String(p.number), '--json', 'comments,labels'])));
  prViews.forEach((r, k) => {
    const pr = openPrs[k];
    const rollup = pr.statusCheckRollup || [];
    const failed = rollup.filter(c => c.conclusion && c.conclusion !== 'success' && c.conclusion !== 'skipped');
    const pending = rollup.filter(c => !c.conclusion);
    state.ci[pr.number] = rollup.length === 0 ? 'none'
      : failed.length ? 'red' : pending.length ? 'pending' : 'green';
    if (!r.ok || !r.data) return;
    const comments = r.data.comments || [];
    state.prComments[pr.number] = comments.slice(-4).map(c => ({
      author: c.author ? c.author.login : '?', at: c.createdAt, body: c.body || '',
    }));
    for (let j = comments.length - 1; j >= 0; j--) {
      const b = comments[j].body || '';
      const m = b.match(/\*\*(GO|NO-GO|NO GO)\*\*/) || b.match(/\b(GO|NO-GO|NO GO)\b/);
      if (m && /revue|verdict|groupe c|review/i.test(b)) {
        state.reviews[pr.number] = {
          verdict: m[1].toUpperCase().replace('NO GO', 'NO-GO'),
          excerpt: b.replace(/[#*`>]/g, '').slice(0, 260),
        };
        break;
      }
    }
  });

  /* Portfolio : contenu réel de projects/<axe>/ (shippé sur main) + présence README */
  state.portfolio = {};
  await Promise.all(AXES.map(async (axe) => {
    const r = await ghApi(`contents/projects/${axe}`);
    if (!r.ok || !Array.isArray(r.data)) { state.portfolio[axe] = []; return; }
    const dirs = r.data.filter(e => e.type === 'dir').slice(0, 5);
    const projects = await Promise.all(dirs.map(async (d) => {
      const rr = await ghApi(`contents/projects/${axe}/${d.name}/README.md`);
      return { slug: d.name, readme: rr.ok };
    }));
    state.portfolio[axe] = projects;
  }));

  return state;
}

/* ---------- Cache (évite de marteler l'API) ---------- */
let cache = { when: 0, state: null };
async function getState() {
  if (cache.state && Date.now() - cache.when < 15000) return cache.state;
  const s = await fetchState();
  if (s.issues && s.prs) cache = { when: Date.now(), state: s };
  return cache.state || s;
}

/* ---------- Actions : uniquement les décisions humaines ---------- */
const ACTIONS = {
  approve_idea: (n) => gh(['issue', 'edit', String(n), '-R', REPO, '--add-label', 'idea/approved']),
  reject_idea: async (n, reason) => {
    await gh(['issue', 'comment', String(n), '-R', REPO, '--body',
      '🚫 Rejetée par l\'humain — raison : ' + String(reason || 'non précisée')]);
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
  /* Le point de communication humain ↔ agents : commentaire d'issue.
     mode 'fix' → préfixé « !fix » → workflow retour-humain → session Aider. */
  note: async (n, extra) => {
    const text = String((extra && extra.text) || '').trim();
    if (!text) return { ok: false, error: 'note vide' };
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
  console.log(`🏭 Dashboard usine v2 : http://localhost:${PORT}  (repo ${REPO})`);
  console.log('   Notes : mode "fix" = !fix → la flotte exécute ; mode "note" = commentaire simple.');
});