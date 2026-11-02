// Tests de motdujour — node:test + node:assert, std lib uniquement.
import { test } from 'node:test';
import assert from 'node:assert/strict';

import { WORDS, todayISO, wordForDate, parseArgs, run } from '../src/index.js';

test('la liste intégrée contient exactement 100 mots uniques et non vides', () => {
  assert.equal(WORDS.length, 100);
  assert.equal(new Set(WORDS).size, 100);
  for (const w of WORDS) {
    assert.ok(w.length > 0, 'mot vide');
    assert.ok(!/\s/.test(w), `espaces dans « ${w} »`);
  }
});

test('wordForDate renvoie toujours un mot de la liste', () => {
  for (const d of ['2025-01-01', '2025-06-15', '1999-12-31', '2024-02-29']) {
    assert.ok(WORDS.includes(wordForDate(d)), `mot hors liste pour ${d}`);
  }
});

test('wordForDate est déterministe : même date, même mot', () => {
  for (const d of ['2025-01-01', '2025-06-15', '1999-12-31', '2024-02-29']) {
    assert.equal(wordForDate(d), wordForDate(d));
  }
});

test('wordForDate varie avec la date : 365 jours donnent au moins 10 mots distincts', () => {
  const words = new Set();
  for (let i = 0; i < 365; i++) {
    words.add(wordForDate(todayISO(new Date(2025, 0, 1 + i))));
  }
  assert.ok(words.size >= 10, `seulement ${words.size} mots distincts`);
});

test('todayISO formate une date locale en YYYY-MM-DD', () => {
  assert.equal(todayISO(new Date(2025, 0, 1)), '2025-01-01');
  assert.equal(todayISO(new Date(2025, 2, 5)), '2025-03-05');
  assert.equal(todayISO(new Date(1999, 11, 31)), '1999-12-31');
});

test('parseArgs : sans argument, mode texte', () => {
  assert.deepEqual(parseArgs([]), { json: false, error: null });
});

test('parseArgs : --json active le mode JSON', () => {
  assert.deepEqual(parseArgs(['--json']), { json: true, error: null });
});

test('parseArgs : option inconnue ou argument parasite, erreur', () => {
  for (const argv of [['--xml'], ['mot'], ['--json', 'mot'], ['mot', '--json']]) {
    const { json, error } = parseArgs(argv);
    assert.ok(error !== null, `pas d'erreur pour ${JSON.stringify(argv)}`);
    assert.equal(json, false);
  }
});

test('parseArgs : --json répété, erreur', () => {
  assert.ok(parseArgs(['--json', '--json']).error !== null);
});

test('run sans argument : code 0, mot du jour sur stdout, rien sur stderr', () => {
  const date = new Date(2025, 4, 17);
  const { code, stdout, stderr } = run([], date);
  assert.equal(code, 0);
  assert.equal(stderr, '');
  const word = stdout.trimEnd();
  assert.ok(WORDS.includes(word));
  assert.equal(word, wordForDate(todayISO(date)));
});

test('run --json : code 0, JSON valide avec date et mot cohérents', () => {
  const { code, stdout, stderr } = run(['--json'], new Date(2025, 4, 17));
  assert.equal(code, 0);
  assert.equal(stderr, '');
  const obj = JSON.parse(stdout);
  assert.equal(obj.date, '2025-05-17');
  assert.equal(obj.word, wordForDate('2025-05-17'));
});

test('run --json sans date injectée : structure du jour cohérente', () => {
  const { code, stdout } = run(['--json']);
  assert.equal(code, 0);
  const obj = JSON.parse(stdout);
  assert.match(obj.date, /^\d{4}-\d{2}-\d{2}$/);
  assert.ok(WORDS.includes(obj.word));
});

test('run avec argument invalide : code 1, erreur sur stderr, stdout vide', () => {
  const { code, stdout, stderr } = run(['--xml']);
  assert.equal(code, 1);
  assert.equal(stdout, '');
  assert.ok(stderr.includes('motdujour'));
});
