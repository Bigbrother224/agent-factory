// motdujour — le mot du jour en français, déterministe par la date.
// Std lib uniquement : node:crypto (SHA-256), node:url (détection d'exécution directe).
import { createHash } from 'node:crypto';
import { pathToFileURL } from 'node:url';

// Liste intégrée de 100 mots français : aucune dépendance, aucun réseau.
export const WORDS = [
  'abricot', 'alizé', 'ancolie', 'aquarelle', 'bonheur',
  'boussole', 'brise', 'brouillard', 'brume', 'buisson',
  'cabane', 'calanque', 'cascade', 'cèdre', 'chandelle',
  'châtaigne', 'clairière', 'colibri', 'coquelicot', 'corolle',
  'cormoran', 'cygne', 'dauphin', 'dune', 'éclair',
  'éclat', 'écorce', 'éphémère', 'étoile', 'estuaire',
  'fenêtre', 'flamme', 'forêt', 'fougère', 'garrigue',
  'gentiane', 'girafe', 'goutte', 'grange', 'grésil',
  'grive', 'haie', 'hermine', 'horizon', 'huppe',
  'île', 'ivoire', 'jasmin', 'kayak', 'lanterne',
  'libellule', 'lichen', 'lisière', 'lotus', 'luciole',
  'marée', 'mélèze', 'miroir', 'mosaïque', 'myrtille',
  'neige', 'nid', 'nuage', 'océan', 'ombre',
  'onde', 'orchidée', 'orage', 'origan', 'palmier',
  'papillon', 'pastel', 'pépite', 'pétale', 'phare',
  'pinède', 'pivoine', 'plume', 'quartz', 'radeau',
  'rameau', 'ravin', 'rivière', 'rossignol', 'roseau',
  'ronce', 'sable', 'saule', 'sentier', 'sieste',
  'sillage', 'soleil', 'thym', 'trèfle', 'tulipe',
  'vallon', 'velours', 'violette', 'vigne', 'zéphyr',
];

// Date locale au format YYYY-MM-DD (le « jour » est celui de l'utilisateur).
export function todayISO(date = new Date()) {
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

// Mot déterministe pour une date YYYY-MM-DD : SHA-256 de la date, modulo la taille de la liste.
export function wordForDate(isoDate) {
  const digest = createHash('sha256').update(isoDate).digest();
  return WORDS[digest.readUInt32BE(0) % WORDS.length];
}

// Arguments acceptés : aucun, ou exactement une fois --json. Tout le reste est une erreur.
export function parseArgs(argv) {
  const unexpected = argv.filter((a) => a !== '--json');
  const jsonCount = argv.length - unexpected.length;
  if (unexpected.length > 0) {
    return { json: false, error: `argument inattendu : « ${unexpected[0]} »` };
  }
  if (jsonCount > 1) {
    return { json: false, error: 'option --json répétée' };
  }
  return { json: jsonCount === 1, error: null };
}

// Exécution pure : renvoie code de sortie et flux, sans toucher au processus (testable).
export function run(argv, date = new Date()) {
  const { json, error } = parseArgs(argv);
  if (error !== null) {
    return { code: 1, stdout: '', stderr: `motdujour : ${error}\n` };
  }
  const iso = todayISO(date);
  const word = wordForDate(iso);
  const stdout = json
    ? `${JSON.stringify({ date: iso, word })}\n`
    : `${word}\n`;
  return { code: 0, stdout, stderr: '' };
}

function main() {
  const { code, stdout, stderr } = run(process.argv.slice(2));
  if (stdout !== '') process.stdout.write(stdout);
  if (stderr !== '') process.stderr.write(stderr);
  process.exitCode = code;
}

// CLI uniquement en exécution directe (node src/index.js), jamais à l'import.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
