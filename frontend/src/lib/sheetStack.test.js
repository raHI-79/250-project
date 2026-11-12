import assert from 'node:assert/strict'
import test from 'node:test'
import { closeTopSheet, openSheets, pushSheet } from './sheetStack.js'

test('the back button closes the sheet on top before leaving the page', () => {
  const closed = []
  assert.equal(closeTopSheet(), false)
  const first = { close: () => closed.push('first') }
  const second = { close: () => closed.push('second') }
  const dropFirst = pushSheet(first)
  const dropSecond = pushSheet(second)
  assert.equal(openSheets(), 2)
  assert.equal(closeTopSheet(), true)
  assert.deepEqual(closed, ['second'])
  dropSecond()
  assert.equal(closeTopSheet(), true)
  assert.deepEqual(closed, ['second', 'first'])
  dropFirst()
  assert.equal(openSheets(), 0)
})

test('a sheet that cannot be dismissed still swallows the back button', () => {
  const drop = pushSheet({})
  assert.equal(closeTopSheet(), true)
  drop()
  assert.equal(closeTopSheet(), false)
})
