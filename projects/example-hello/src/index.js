export function greet(name) {
  if (typeof name !== 'string' || name.trim() === '') {
    throw new Error('name requis (chaîne non vide)');
  }
  return `Hello, ${name.trim()}!`;
}