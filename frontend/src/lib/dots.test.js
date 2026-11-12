import assert from 'node:assert/strict'
import test from 'node:test'
import { dots } from './dots.js'

test('an empty field leaves no stray dot behind', () => {
  assert.equal(dots('CSE245', 'Databases'), 'CSE245 · Databases')
  assert.equal(dots('CSE245', ''), 'CSE245')
  assert.equal(dots('', 'CSE-23', null, undefined, '  ', 'Section A'), 'CSE-23 · Section A')
  assert.equal(dots(['a', ['', 'b']]), 'a · b')
  assert.equal(dots(), '')
  assert.equal(dots(0, 'x'), '0 · x')
})
