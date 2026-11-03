#!/usr/bin/env node
/**
 * Dashboard de l'usine à idées — agent-factory
 * Sert localhost:7777 : vue live des issues/PRs/runs GitHub + les 2 portes cliquables.
 * Zéro dépendance. Le repo GitHub reste l'unique bus de communication :
 * ce dashboard ne fait que lire (gh api) et exécuter tes décisions (gh label/merge).
 */
const http = require('http');
const { execFile } = require('child_process');
const fs = require('fs');
const path = require('path');

const REPO = process.env.FACTORY_REPO || 'Bigbrother224/agent-factory';
const PORT = parseInt(process.env.PORT || '7777', 10);
const HTML = path.join(__dirname, 'dashboard.html');

function gh(args) {
  return new Promise((resolve) => {
    execFile('gh', args, { timeout: 20000, maxBuffer: 10 * 1024 * 1024 }, (err, stdout, stderr) => {
      resolve({ ok: !err, out: stdout, err: stderr || (err && err.message) || '' });
    });
  });
}

async function ghJson(args) {
  const r = await gh([...args, '-R', REPO]);
  if (!r.ok) return { ok: false, data: null, err: r.err };
  try { return { ok: true, data: JSON.parse(r.out), err: '' }; }
  catch (e) { return { ok: false, data: null, err: 'JSON invalide: ' + e.message }; }
}

async function fetchState() {
  const [issues, prs, runs] = await Promise.all([
    ghJson(['issue', 'list', '--state', 'all', '--limit', '50', '--json',
      'number,title,state,labels,updatedAt,url']),
    ghJson(['pr', 'list', '--state', 'open', '--limit', '20', '--json',
      'number,title,state,labels,headRefName,statusCheckRollup,updatedAt,url,author']),
    ghJson(['run', 'list', '--limit', '25', '--json',
      'databaseId,displayTitle,workflowName,status,conclusion,createdAt,event,headBranch']),
  ]);

  const state = { repo: REPO, when: new Date().toISOString(), errors: [] };
  if (issues.ok) state.issues = issues.data; else state.errors.push('issues: ' + issues.err);
  if (prs.ok) state.prs = prs.data; else state.errors.push('prs: ' + prs.err);
  if (runs.ok) state.runs = runs.data; else state.errors.push('runs: ' + runs.err);

  // Revue kimi (GO/NO-GO) : extraite des commentaires de chaque PR ouverte (max 8)
  state.reviews = {};
  if (state.prs) {
    const prsToCheck = state.prs.slice(0, 8);
    const results = await Promise.all(prsToCheck.map((pr) =>
      ghJson(['pr', 'view', String(pr.number), '--json', 'comments'])));
    results.forEach((r, i) => {
      if (!r.ok || !r.data || !Array.isArray(r.data.comments)) return;
      const pr = prsToCheck[i];
      // cherche le dernier commentaire kimi avec verdict
      for (let j = r.data.comments.length - 1; j >= 0; j--) {
        const body = r.data.comments[j].body || '';
        const m = body.match(/\b(GO|NO-GO|NO GO)\b/i);
        if (m && /kimi|revue|review|vérif/i.test(body + r.data.comments[j].author?.login || '')) {
          state.reviews[pr.number] = {
            verdict: m[1].toUpperCase().replace('NO GO', 'NO-GO'),
            excerpt: body.slice(0, 300),
          };
          return;
        }
      }
    });
  }

  // CI par PR (depuis statusCheckRollup)
  state.ci = {};
  if (state.prs) {
    for (const pr of state.prs) {
      const rollup = pr.statusCheckRollup || [];
      const failed = rollup.filter(c => c.conclusion && c.conclusion !== 'success' && c.conclusion !== 'skipped');
      const pending = rollup.filter(c => !c.conclusion);
      state.ci[pr.number] = rollup.length === 0 ? 'none'
        : failed.length ? 'red'
        : pending.length ? 'pending'
        : 'green';
    }
  }
  return state;
}

// Cache pour éviter de marteler l'API GitHub
let cache = { when: 0, state: null };
async function getState() {
  if (cache.state && Date.now() - cache.when < 8000) return cache.state;
  const s = await fetchState();
  if (s.issues || s.prs || s.runs) cache = { when: Date.now(), state: s };
  return cache.state || s;
}

// ---- Actions (tes 2 portes + management de base) ----
const ACTIONS = {
  approve_idea: (n) => ['issue', 'edit', String(n), '--add-label', 'idea/approved'],
  reject_idea: (n) => ['issue', 'edit', String(n), '--remove-label', 'idea/draft'],
  close_issue: (n) => ['issue', 'close', String(n), '--reason', 'not planned'],
  merge_pr: (n) => ['pr', 'merge', String(n), '--merge', '--delete-branch'],
  close_pr: (n) => ['pr', 'close', String(n)],
  comment_pr: (n, body) => ['pr', 'comment', String(n), '--body', String(body || '').slice(0, 2000)],
  rerun: (id) => ['run', 'rerun', String(id)],
};

async function doAction(type, target, extra) {
  const factory = ACTIONS[type];
  if (!factory) return { ok: false, error: 'Action inconnue: ' + type };
  const r = await gh([...factory(target, extra), '-R', REPO]);
  cache = { when: 0, state: null }; // invalide le cache
  return { ok: r.ok, error: r.ok ? '' : r.err };
}

const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && (req.url === '/' || req.url.startsWith('/index'))) {
    res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' });
    res.end(fs.readFileSync(HTML));
  } else if (req.method === 'GET' && req.url.startsWith('/api/state')) {
    const state = await getState();
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end(JSON.stringify(state));
  } else if (req.method === 'POST' && req.url.startsWith('/api/action')) {
    let body = '';
    req.on('data', (c) => { body += c; if (body.length > 10000) req.destroy(); });
    req.on('end', async () => {
      try {
        const { type, target, extra } = JSON.parse(body || '{}');
        const targetNum = parseInt(target, 10);
        if (isNaN(targetNum) && type !== 'comment_pr') throw new Error('cible invalide');
        const r = await doAction(type, targetNum, extra);
        res.writeHead(r.ok ? 200 : 500, { 'content-type': 'application/json' });
        res.end(JSON.stringify(r));
      } catch (e) {
        res.writeHead(400, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ ok: false, error: e.message }));
      }
    });
  } else {
    res.writeHead(404); res.end('not found');
  }
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`🏭 Dashboard usine: http://localhost:${PORT}  (repo ${REPO})`);
  console.log('   Portes: cliquer sur la page exécute gh label/merge en local.');
});