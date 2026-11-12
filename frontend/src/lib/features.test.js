import assert from 'node:assert/strict'
import test from 'node:test'
import {
  FEATURE_ATTENDANCE,
  FEATURE_GROUPS,
  FEATURE_MESSAGES,
  FEATURE_POLLS,
  featureOffMessage,
  featureOfPage,
  featureOfRequest,
  featureOn,
  pageOn,
  requestOn,
  visibleGroups,
  visibleItems,
} from './features.js'

const messagesOff = { messages: false, polls: true, attendance: true, group_chats: true }

test('a feature is on until the config says it is off', () => {
  assert.equal(featureOn(undefined, FEATURE_MESSAGES), true)
  assert.equal(featureOn({}, FEATURE_MESSAGES), true)
  assert.equal(featureOn(messagesOff, FEATURE_MESSAGES), false)
  assert.equal(featureOn(messagesOff, FEATURE_POLLS), true)
  assert.equal(featureOn(messagesOff, ''), true)
})

test('pages and requests know which switch they belong to', () => {
  assert.equal(featureOfPage('/student/messages?c=4'), FEATURE_MESSAGES)
  assert.equal(featureOfPage('/teacher/messages'), FEATURE_MESSAGES)
  assert.equal(featureOfPage('/cr/polls?highlight=2'), FEATURE_POLLS)
  assert.equal(featureOfPage('/student/my-attendance'), FEATURE_ATTENDANCE)
  assert.equal(featureOfPage('/teacher/phone-requests'), FEATURE_ATTENDANCE)
  assert.equal(featureOfPage('/student/notices'), '')
  assert.equal(featureOfPage('/office-portal/messages'), '')
  assert.equal(featureOfRequest('/api/messages/unread-count'), FEATURE_MESSAGES)
  assert.equal(featureOfRequest('/api/messages/conversations'), FEATURE_MESSAGES)
  assert.equal(featureOfRequest('/api/messages/contacts?search=a'), FEATURE_MESSAGES)
  assert.equal(featureOfRequest('/api/students/polls'), FEATURE_POLLS)
  assert.equal(featureOfRequest('/api/students/attendance/mine'), FEATURE_ATTENDANCE)
  assert.equal(featureOfRequest('/api/students/notices'), '')
})

test('a switched-off feature is neither called nor shown', () => {
  assert.equal(requestOn(messagesOff, '/api/messages/unread-count'), false)
  assert.equal(requestOn(messagesOff, '/api/students/polls'), true)
  assert.equal(pageOn(messagesOff, '/student/messages'), false)
  const items = [
    { to: '/student/today', label: 'Today' },
    { to: '/student/messages', label: 'Messages' },
    { to: '/student/polls', label: 'Polls' },
  ]
  assert.deepEqual(
    visibleItems(messagesOff, items).map((item) => item.label),
    ['Today', 'Polls'],
  )
  assert.equal(visibleItems(undefined, items).length, 3)
  const built = [{ to: () => '/student/messages' }, { to: () => '/student/today' }]
  assert.equal(visibleItems(messagesOff, built).length, 1, 'a navigation item built by a function stayed')
  const groups = [
    { label: 'Talk', items: [{ to: '/student/messages' }] },
    { label: 'Class', items: [{ to: '/student/today' }, { to: '/student/polls' }] },
  ]
  const shown = visibleGroups({ messages: false, polls: false }, groups)
  assert.equal(shown.length, 1)
  assert.deepEqual(
    shown[0].items.map((item) => item.to),
    ['/student/today'],
  )
})

test('a direct visit says which feature the department turned off', () => {
  assert.equal(featureOffMessage(FEATURE_MESSAGES), 'Messages is turned off by the department')
  assert.equal(featureOffMessage(FEATURE_POLLS), 'Polls is turned off by the department')
  assert.equal(featureOffMessage(FEATURE_ATTENDANCE), 'Attendance is turned off by the department')
  assert.equal(featureOffMessage(FEATURE_GROUPS), 'Group chats is turned off by the department')
})
