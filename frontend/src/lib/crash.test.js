import assert from 'node:assert/strict'
import test from 'node:test'
import { crashReport, crashRoute, createCrashReporter } from './crash.js'

test('a crash report carries the name, message and route and nothing personal', () => {
  const error = new TypeError("Cannot read properties of null (reading 'batch')")
  const report = crashReport(
    error,
    { pathname: '/student/upcoming-routine', search: '?date=2026-11-14&email=a@b.c' },
    'app',
    '2.7.3',
  )
  assert.deepEqual(Object.keys(report).sort(), ['client', 'message', 'name', 'route', 'version'])
  assert.equal(report.name, 'TypeError')
  assert.equal(report.route, '/student/upcoming-routine')
  assert.equal(report.client, 'app')
  assert.ok(!JSON.stringify(report).includes('a@b.c'))
})

test('a crash route never keeps a query or a fragment', () => {
  assert.equal(crashRoute({ pathname: '/student/notices?open=4#top' }), '/student/notices')
  assert.equal(crashRoute(null), '/')
  assert.equal(crashRoute({ pathname: 'profile' }), '/profile')
})

test('the same crash is reported once', () => {
  const sent = []
  const report = createCrashReporter((body) => {
    sent.push(body)
    return Promise.resolve()
  })
  const body = crashReport(new Error('boom'), { pathname: '/profile' }, 'web', '2.7.3')
  assert.equal(report(body), true)
  assert.equal(report(body), false)
  assert.equal(report({ ...body, route: '/student/today' }), true)
  assert.equal(sent.length, 2)
})

test('a reporter that cannot send never throws', () => {
  const report = createCrashReporter(() => {
    throw new Error('offline')
  })
  assert.equal(report(crashReport(new Error('boom'), { pathname: '/' }, 'web', '1')), false)
})
