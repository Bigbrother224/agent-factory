import { test } from 'node:test';
import assert from 'node:assert/strict';
import { greet } from '../src/index.js';

test('salue une personne', () => {
  assert.equal(greet('Factory'), 'Hello, Factory!');
});

test('trimme le nom avant de saluer', () => {
  assert.equal(greet('  Factory  '), 'Hello, Factory!');
});

test('salue un nom accenté', () => {
  assert.equal(greet('Élise'), 'Hello, Élise!');
});

test('refuse un nom vide', () => {
  assert.throws(() => greet(''));
});

test('refuse un nom fait d\'espaces', () => {
  assert.throws(() => greet('   '));
});

test('refuse un non-chaîne (nombre)', () => {
  assert.throws(() => greet(42));
});

test('refuse un non-chaîne (null)', () => {
  assert.throws(() => greet(null));
});

test('lève une Error dont le message mentionne « name »', () => {
  assert.throws(() => greet(''), /name/);
});