import { test } from 'node:test';
import assert from 'node:assert/strict';
import { greet } from '../src/index.js';

test('salue une personne', () => {
  assert.equal(greet('Factory'), 'Hello, Factory!');
});

test('refuse un nom vide', () => {
  assert.throws(() => greet('  '));
});

// Volontairement défaillant : preuve que l'arbitre refuse les PR rouges.
test('garantia volontairement fausse', () => {
  assert.equal(greet('Factory'), 'Bonjour le monde');  // attendu ≠ réel
});
