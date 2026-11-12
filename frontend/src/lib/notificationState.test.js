import assert from 'node:assert/strict'
import test from 'node:test'
import {
  EMPTY_NOTIFICATIONS,
  badgeLabel,
  pointsAtAnItem,
  received,
  samePage,
  unreadOnPage,
  withAllRead,
  withRead,
  withUnread,
} from './notificationState.js'

const sample = () =>
  received({
    unread: 3,
    items: [
      { id: 1, read: false, link: { kind: 'notice', ref: 7 } },
      { id: 2, read: false, link: { kind: 'routine', date: '2026-11-14' } },
      { id: 3, read: true, link: { kind: 'notice', ref: 7 } },
      { id: 4, read: false, link: { kind: 'notice' } },
    ],
  })

test('the badge shows the exact number up to 99 and 99+ above', () => {
  assert.equal(badgeLabel(0), '')
  assert.equal(badgeLabel(-2), '')
  assert.equal(badgeLabel(1), '1')
  assert.equal(badgeLabel(10), '10')
  assert.equal(badgeLabel(99), '99')
  assert.equal(badgeLabel(100), '99+')
  assert.equal(badgeLabel(4821), '99+')
})

test('opening one notification lowers the count by one at once', () => {
  const before = sample()
  const after = withRead(before, [1])
  assert.equal(after.unread, 2)
  assert.equal(after.items[0].read, true)
  assert.equal(before.items[0].read, false, 'the state before was changed, so it cannot be rolled back')
  assert.equal(withRead(after, [1]), after, 'reading it twice lowered the count twice')
  assert.equal(withRead(after, [3]), after, 'a read notification lowered the count')
})

test('the count never goes below zero and follows the server', () => {
  assert.equal(withRead({ ...sample(), unread: 0 }, [1]).unread, 0)
  assert.equal(withUnread(sample(), 12).unread, 12)
  assert.equal(withUnread(sample(), undefined).unread, 3)
  assert.equal(withAllRead(sample()).unread, 0)
  assert.equal(
    withAllRead(sample()).items.every((item) => item.read),
    true,
  )
  assert.equal(received(null).unread, 0)
  assert.equal(received({ items: [{ id: 1 }, { id: 2, read: true }] }).unread, 1)
  assert.equal(EMPTY_NOTIFICATIONS.loaded, false)
})

test('a page matches the notifications that point at it', () => {
  assert.equal(samePage('/student/notices/7', '/student/notices/7'), true)
  assert.equal(samePage('/student/notices/7/', '/student/notices/7'), true)
  assert.equal(samePage('/student/notices/8', '/student/notices/7'), false)
  assert.equal(samePage('/student/exams?highlight=4&x=1', '/student/exams?highlight=4'), true)
  assert.equal(samePage('/student/exams', '/student/exams?highlight=4'), false)
  const targetOf = (item) => {
    if (item.link.kind !== 'notice') return '/student/upcoming-routine?date=' + item.link.date
    return item.link.ref ? '/student/notices/' + item.link.ref : '/student/notices'
  }
  assert.deepEqual(unreadOnPage(sample(), '/student/notices/7', targetOf), [1])
  assert.deepEqual(unreadOnPage(sample(), '/student/upcoming-routine?date=2026-11-14', targetOf), [2])
  assert.deepEqual(unreadOnPage(sample(), '/student/notices', targetOf), [])
})

test('only a page that names an item is reported as followed', () => {
  assert.equal(pointsAtAnItem('/student/notices/7'), true)
  assert.equal(pointsAtAnItem('/student/exams?highlight=4'), true)
  assert.equal(pointsAtAnItem('/student/messages?c=9'), true)
  assert.equal(pointsAtAnItem('/student/notices'), false)
  assert.equal(pointsAtAnItem('/cr/create-class?tab=final'), false)
})
