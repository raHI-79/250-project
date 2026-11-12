import { expect, test as base } from '@playwright/test'
import {
  CLASS_EARLY,
  CLASS_LATE,
  DATE_PAGES,
  DATE_PANELS,
  PHONE_PAGES,
  ROLES,
} from '../fixtures/roles.js'
import { clear, expectNoProblems, watch } from '../helpers/audit.js'
import { appCanReach, asApp, localURL } from '../helpers/appShell.js'
import { call, classOf, clearToday, need, signInByApi, workbookEntry } from '../helpers/attendance.js'
import { exerciseDateFields, setTheme } from '../helpers/dates.js'
import { postNotice, removeNotice } from '../helpers/notices.js'
import { signIn, visit } from '../helpers/signIn.js'
import {
  DIALOG,
  closeDialog,
  collectOpeners,
  expectDialogWhole,
  expectPageScrollsAgain,
  expectPageWhole,
} from '../helpers/views.js'

const test = base.extend({ appShell: [false, { option: true }] })

const THEMES = ['light', 'dark']

const settled = async (page) => {
  await page.waitForLoadState('networkidle', { timeout: 10_000 }).catch(() => {})
  await page.waitForTimeout(250)
}

async function openOne(page, path, opener, where, problems) {
  const target = page.locator(`[data-e2e-opener="${opener.index}"]`)
  if ((await target.count()) === 0) return 0
  const before = page.url()
  clear(problems)
  await target.scrollIntoViewIfNeeded().catch(() => {})
  await target.click({ timeout: 8_000 }).catch(() => {})
  await page.waitForTimeout(400)
  const opened = await page.locator(DIALOG).count()
  if (opened > 0) {
    await expectDialogWhole(page, opener, where)
    expectNoProblems(problems, where)
    await closeDialog(page)
    await expectPageScrollsAgain(page, `${where} after closing`)
    return 1
  }
  if (page.url() !== before) {
    await settled(page)
    expectNoProblems(problems, where)
    await expectPageWhole(page, where)
    await page.goBack({ waitUntil: 'domcontentloaded' }).catch(() => {})
    await settled(page)
    if (!page.url().includes(path.split('?')[0])) await visit(page, path)
    return 1
  }
  return 0
}

async function openTheMenu(page, where) {
  const menu = page.getByRole('button', { name: 'Open menu' }).first()
  if ((await menu.count()) === 0 || !(await menu.isVisible())) return 0
  await menu.click()
  await expect(page.locator(DIALOG)).toHaveCount(1)
  await expectDialogWhole(page, { kind: 'button', name: 'Open menu' }, `${where}: the menu`)
  await closeDialog(page)
  await expectPageScrollsAgain(page, `${where}: after the menu`)
  return 1
}

async function openNotifications(page, where) {
  const bell = page.getByRole('button', { name: 'Notifications' }).first()
  if ((await bell.count()) === 0 || !(await bell.isVisible())) return 0
  await bell.click()
  await expect(page.locator(DIALOG)).toHaveCount(1)
  await expectDialogWhole(page, { kind: 'button', name: 'Notifications' }, `${where}: notifications`)
  await closeDialog(page)
  await expectPageScrollsAgain(page, `${where}: after notifications`)
  return 1
}

for (const [key, role] of Object.entries(ROLES)) {
  test.describe(`the ${role.label} on a phone`, () => {
    test('opens every view whole, scrollable and closable', async ({ page, appShell }, info) => {
      test.skip(appShell && !appCanReach(key), 'the office portal and control room are website only')
      test.setTimeout(45 * 60_000)
      if (appShell) await asApp(page)
      const problems = watch(page)
      await signIn(page, role)
      clear(problems)
      problems.signedIn = true
      const shell = appShell ? 'app' : 'website'
      const width = page.viewportSize()?.width
      let views = 0
      views += await openTheMenu(page, `${shell} ${width}px as the ${role.label}`)
      views += await openNotifications(page, `${shell} ${width}px as the ${role.label}`)
      for (const path of PHONE_PAGES[key]) {
        await visit(page, path)
        const where = `${path} (${shell}, ${width}px, ${role.label})`
        expectNoProblems(problems, where)
        await expectPageWhole(page, where)
        await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }))
        const openers = await collectOpeners(page)
        for (const opener of openers) {
          if (!page.url().includes(path.split('?')[0])) await visit(page, path)
          const fresh = await collectOpeners(page)
          const same = fresh.find((entry) => entry.index === opener.index)
          if (!same) continue
          views += await openOne(page, path, same, `"${same.name}" on ${where}`, problems)
        }
      }
      info.annotations.push({ type: 'views', description: String(views) })
      console.log(`${shell} ${width}px ${role.label}: ${views} views opened`)
      expect(views, `the ${role.label} opened nothing`).toBeGreaterThan(0)
    })

    test('taps every date field in light and dark without losing the page', async ({ page, appShell }) => {
      test.skip(appShell && !appCanReach(key), 'the office portal and control room are website only')
      test.setTimeout(20 * 60_000)
      if (appShell) await asApp(page)
      const problems = watch(page)
      await signIn(page, role)
      clear(problems)
      problems.signedIn = true
      const shell = appShell ? 'app' : 'website'
      let fields = 0
      for (const theme of THEMES) {
        await setTheme(page, theme)
        for (const path of DATE_PAGES[key]) {
          await visit(page, path)
          clear(problems)
          fields += await exerciseDateFields(page, problems, `${path} (${shell}, ${theme}, ${role.label})`)
        }
      }
      expect(fields, `the ${role.label} found no date field`).toBeGreaterThan(0)
    })
  })
}

test.describe('the sign-up panels on a phone', () => {
  test('tap every date field in light and dark without losing the page', async ({ page, appShell }) => {
    if (appShell) await asApp(page)
    const problems = watch(page)
    for (const theme of THEMES) {
      await setTheme(page, theme)
      for (const path of DATE_PANELS) {
        await page.goto(path, { waitUntil: 'domcontentloaded' })
        await settled(page)
        clear(problems)
        await exerciseDateFields(page, problems, `${path} (${theme})`)
      }
    }
  })
})

test.describe('a screen that breaks on a phone', () => {
  test('shows a calm screen, reports it and recovers', async ({ page, appShell }) => {
    if (appShell) await asApp(page)
    await signIn(page, ROLES.student)
    const routine = '**/api/students/upcoming-routine*'
    await page.route(routine, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ batch: 'CSE', date_label: 'A day', weekday: 'Sunday', groups: 5 }),
      }),
    )
    const reported = page.waitForRequest(
      (request) => request.url().endsWith('/api/client-log') && request.method() === 'POST',
      { timeout: 20_000 },
    )
    await page.goto('/student/upcoming-routine', { waitUntil: 'domcontentloaded' })
    const screen = page.locator('[data-crash-screen]')
    await expect(screen).toBeVisible({ timeout: 20_000 })
    await expect(screen).toContainText('Something went wrong')
    await expect(screen.getByRole('button', { name: 'Go home' })).toBeVisible()
    const body = (await reported).postDataJSON()
    expect(Object.keys(body).sort()).toEqual(['client', 'message', 'name', 'route', 'version'])
    expect(body.route).toBe('/student/upcoming-routine')
    expect(body.client).toBe(appShell ? 'app' : 'web')
    await expect(page.getByRole('button', { name: 'Open menu' }).first()).toBeVisible()
    await page.unroute(routine)
    await screen.getByRole('button', { name: 'Try again' }).tap()
    await expect(screen).toHaveCount(0)
    await expect(page.locator('main h1')).toContainText('Upcoming routine')
    await page.route(routine, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ batch: 'CSE', date_label: 'A day', weekday: 'Sunday', groups: 5 }),
      }),
    )
    await page.reload({ waitUntil: 'domcontentloaded' })
    await expect(screen).toBeVisible({ timeout: 20_000 })
    await page.unroute(routine)
    await screen.getByRole('button', { name: 'Go home' }).tap()
    await expect(page).toHaveURL(/\/student\/today/)
    await expect(screen).toHaveCount(0)
  })
})

test.describe('notice links on a phone', () => {
  test('a home notice card and a notification open that notice and lower the bell', async ({ page, appShell }) => {
    test.setTimeout(5 * 60_000)
    if (appShell) await asApp(page)
    const title = `Phone suite notice ${Date.now()}`
    const notice = await postNotice(title)
    try {
      await signIn(page, ROLES.student)
      await visit(page, '/student/today')
      const card = page.locator(`[data-notice-card="${notice.id}"]`)
      await card.scrollIntoViewIfNeeded()
      await card.tap()
      await expect(page).toHaveURL(new RegExp(`/student/notices/${notice.id}$`))
      await expect(page.locator(`[data-notice="${notice.id}"]`)).toContainText(title)
      await expect(page.locator('[data-notice]')).toHaveCount(1)
      const bell = page.getByRole('button', { name: 'Notifications' }).first()
      const count = async () => {
        const badge = bell.locator('[aria-label$=" unread"]')
        if ((await badge.count()) === 0) return 0
        return Number.parseInt((await badge.getAttribute('aria-label')) || '0', 10)
      }
      await expect.poll(count, { timeout: 15_000 }).toBe(await count())
      const afterCard = await count()
      await bell.tap()
      const row = page.locator('[data-notification]').filter({ hasText: title }).first()
      await expect(row).toHaveAttribute('data-unread', 'false')
      await page.keyboard.press('Escape')
      const second = `Phone suite notice ${Date.now()} again`
      const other = await postNotice(second)
      try {
        await page.reload({ waitUntil: 'domcontentloaded' })
        await expect.poll(count, { timeout: 20_000 }).toBe(afterCard + 1)
        await bell.tap()
        const fresh = page.locator('[data-notification]').filter({ hasText: second }).first()
        await expect(fresh).toHaveAttribute('data-unread', 'true')
        await fresh.tap()
        await expect(page).toHaveURL(new RegExp(`/student/notices/${other.id}$`))
        await expect(page.locator(`[data-notice="${other.id}"]`)).toContainText(second)
        expect(await count()).toBe(afterCard)
        await page.reload({ waitUntil: 'domcontentloaded' })
        await expect(page.locator(`[data-notice="${other.id}"]`)).toContainText(second)
        await page.waitForLoadState('networkidle').catch(() => {})
        expect(await count()).toBe(afterCard)
      } finally {
        await removeNotice(other)
      }
      await page.reload({ waitUntil: 'domcontentloaded' })
      await expect(page.getByText('This notice is no longer available')).toBeVisible()
      await page.getByRole('link', { name: 'All notices' }).tap()
      await expect(page).toHaveURL(/\/student\/notices$/)
      await page.goto(`/student/notices?highlight=${notice.id}`, { waitUntil: 'domcontentloaded' })
      await expect(page).toHaveURL(new RegExp(`/student/notices/${notice.id}$`))
      await expect(page.locator(`[data-notice="${notice.id}"]`)).toContainText(title)
    } finally {
      await removeNotice(notice)
    }
  })
})

test.describe('a switched-off feature on a phone', () => {
  test('is neither called nor shown, and a direct visit says so', async ({ page, appShell }) => {
    if (appShell) await asApp(page)
    await page.route('**/api/auth/config', async (route) => {
      const response = await route.fetch({ url: localURL(route.request().url()) })
      const body = await response.json()
      body.features = { ...(body.features || {}), messages: false }
      await route.fulfill({ response, json: body })
    })
    const called = []
    page.on('request', (request) => {
      if (new URL(request.url()).pathname.startsWith('/api/messages')) called.push(request.url())
    })
    const problems = watch(page)
    await signIn(page, ROLES.student)
    clear(problems)
    problems.signedIn = true
    await visit(page, '/student/today')
    await expect(page.locator('a[href="/student/messages"]')).toHaveCount(0)
    await page.goto('/student/messages', { waitUntil: 'domcontentloaded' })
    const off = page.locator('[data-feature-off="messages"]')
    await expect(off).toBeVisible()
    await expect(off).toContainText('Messages is turned off by the department')
    await page.waitForTimeout(1500)
    await visit(page, '/student/polls')
    await expect(page.locator('[data-feature-off]')).toHaveCount(0)
    const afterConfig = called.length
    await page.waitForTimeout(2500)
    expect(called.length, `a switched-off feature was still called: ${called.join(', ')}`).toBe(afterConfig)
    expect(afterConfig, 'the first load polled a switched-off feature').toBeLessThanOrEqual(2)
    expectNoProblems({ ...problems, api: problems.api.filter((line) => !line.includes('/api/messages')) }, 'a switched-off feature')
  })
})

test.describe('attendance rules on a phone', () => {
  test('a roll call takes P, L and A with a custom time and class hours, and a late joiner is absent', async ({
    page,
    appShell,
  }) => {
    test.skip(appShell, 'the app takes attendance by Bluetooth, the roll call is checked on the website')
    test.setTimeout(8 * 60_000)
    const teacher = await signInByApi(ROLES.teacher, 'teacher')
    const student = await signInByApi(CLASS_EARLY, 'student')
    const me = await need(student, 'GET', '/api/profile')
    const entry = await classOf(teacher, me.batch, me.section)
    await clearToday(teacher, entry)
    const joiner = await signInByApi(ROLES.student, 'student')
    const joinerProfile = await need(joiner, 'GET', '/api/profile')
    expect(joinerProfile.batch, 'the late joiner must come from another batch').not.toBe(me.batch)
    let sessionId = 0
    try {
      await signIn(page, ROLES.teacher)
      await visit(page, '/teacher/attendance')
      await page.locator('#attendance-class').selectOption(`${entry.course_id}|${entry.batch}|${entry.section}`)
      await page.getByRole('tab', { name: 'Roll call' }).tap()
      await page.getByRole('radio', { name: 'Custom' }).tap()
      await page.getByLabel('Custom minutes').fill('0')
      await expect(page.getByRole('button', { name: /Start attendance/ })).toBeDisabled()
      await page.getByLabel('Custom minutes').fill('47')
      await page.getByRole('radio', { name: '2 hours' }).tap()
      const started = page.waitForResponse(
        (response) =>
          response.url().endsWith('/api/teacher/attendance/sessions') &&
          response.request().method() === 'POST',
      )
      await page.getByRole('button', { name: /Start attendance/ }).tap()
      const session = await (await started).json()
      sessionId = session.id
      expect(session.hours).toBe(2)
      const minutes = Math.round((new Date(session.ends_at) - new Date(session.started_at)) / 60_000)
      expect(minutes, 'a custom duration of 47 minutes').toBe(47)
      const rows = page.locator('[data-student-row]')
      await expect(rows.first()).toBeVisible()
      const total = await rows.count()
      expect(total, 'the roster needs three students').toBeGreaterThanOrEqual(3)
      const picks = ['late', 'present', 'absent']
      for (let index = 0; index < picks.length; index += 1) {
        const button = rows.nth(index).locator(`[data-mark="${picks[index]}"]`)
        const box = await button.boundingBox()
        expect(Math.min(box.width, box.height), 'a mark button is at least 44 px').toBeGreaterThanOrEqual(44)
        await button.tap()
        await expect(button).toHaveAttribute('aria-pressed', 'true')
      }
      await expect(rows.nth(0).locator('[data-mark="present"]')).toHaveAttribute('aria-pressed', 'false')
      await page.getByRole('button', { name: 'Finish' }).first().tap()
      await page.locator(DIALOG).getByRole('button', { name: 'Finish' }).tap()
      await expect(page.locator('[data-student-row]')).toHaveCount(0, { timeout: 15_000 })
      const detail = await need(teacher, 'GET', `/api/teacher/attendance/sessions/${sessionId}`)
      const counts = { present: 0, late: 0, absent: 0 }
      detail.marks.forEach((mark) => {
        counts[mark.status] += 1
      })
      expect(counts.late).toBe(1)
      expect(counts.present).toBe(1)
      expect(counts.absent).toBe(total - 2)
      const query = `course_id=${entry.course_id}&batch=${entry.batch}&section=${entry.section}`
      const strings = await workbookEntry(teacher, query, 'xl/sharedStrings.xml')
      for (const text of ['Class hours', 'Classes held', 'Attended', 'Lates', 'Percentage']) {
        expect(strings, `the export has no ${text}`).toContain(text)
      }
      expect(strings).toMatch(/<t[^>]*>L<\/t>/)
      const sheet = await workbookEntry(teacher, query, 'xl/worksheets/sheet1.xml')
      expect(sheet, 'the export carries the 2 class hours').toMatch(/<v>2<\/v>/)
      const form = new FormData()
      form.append('drop_course_ids', String(entry.course_id))
      await need(joiner, 'PATCH', '/api/profile', form)
      const mine = await need(joiner, 'GET', '/api/students/attendance/mine')
      const earlier = mine.classes.find((row) => row.session_id === sessionId)
      expect(earlier, 'the earlier class does not count for the late joiner').toBeTruthy()
      expect(earlier.status).toBe('absent')
      expect(earlier.note).toBe('Joined later')
      const course = mine.courses.find((row) => row.course_id === entry.course_id)
      expect(course.held).toBeGreaterThanOrEqual(2)
      expect(course.percentage).toBeLessThan(100)
      const roster = await need(teacher, 'GET', `/api/teacher/attendance/sessions/${sessionId}`)
      const listed = roster.marks.find((mark) => mark.reg_no === joinerProfile.reg_no)
      expect(listed, 'the late joiner is missing from the roster of the earlier class').toBeTruthy()
      await need(teacher, 'PATCH', `/api/teacher/attendance/sessions/${sessionId}/students/${listed.student_id}`, {
        status: 'late',
        note: 'Was in the class',
      })
      const after = await need(joiner, 'GET', '/api/students/attendance/mine')
      expect(after.classes.find((row) => row.session_id === sessionId).status).toBe('late')
    } finally {
      const empty = new FormData()
      empty.append('drop_course_ids', '')
      await call(joiner, 'PATCH', '/api/profile', empty)
      if (sessionId) await call(teacher, 'DELETE', `/api/teacher/attendance/sessions/${sessionId}`)
    }
  })

  test('a mark after an extension is late and shows as Late to the student', async ({ page, appShell }) => {
    test.skip(appShell, 'the website marks with the code, which is what this case uses')
    test.setTimeout(6 * 60_000)
    const teacher = await signInByApi(ROLES.teacher, 'teacher')
    const early = await signInByApi(CLASS_EARLY, 'student')
    const late = await signInByApi(CLASS_LATE, 'student')
    const me = await need(late, 'GET', '/api/profile')
    const entry = await classOf(teacher, me.batch, me.section)
    await clearToday(teacher, entry)
    let sessionId = 0
    try {
      const base = { course_id: entry.course_id, batch: entry.batch, section: entry.section }
      const refused = await call(teacher, 'POST', '/api/teacher/attendance/sessions', {
        ...base,
        minutes: 10,
        late_after_minutes: 11,
        method: 'bluetooth',
        code_only: true,
      })
      expect(refused.status, 'late after later than the end must be refused').toBe(400)
      const tooLong = await call(teacher, 'POST', '/api/teacher/attendance/sessions', {
        ...base,
        minutes: 241,
        method: 'bluetooth',
        code_only: true,
      })
      expect(tooLong.status).toBe(400)
      const session = await need(teacher, 'POST', '/api/teacher/attendance/sessions', {
        ...base,
        minutes: 1,
        hours: 1,
        method: 'bluetooth',
        code_only: true,
      })
      sessionId = session.id
      const longer = await need(teacher, 'POST', `/api/teacher/attendance/sessions/${sessionId}/extend`, {
        minutes: 7,
      })
      expect(Math.round((new Date(longer.ends_at) - new Date(longer.started_at)) / 60_000)).toBe(8)
      const first = await need(early, 'POST', '/api/students/attendance/mark', {
        session_id: sessionId,
        code: longer.code,
      })
      expect(first.status).toBe('present')
      const wait = new Date(longer.original_ends_at).getTime() - Date.now() + 1500
      await page.waitForTimeout(Math.max(wait, 0))
      const now = await need(teacher, 'GET', `/api/teacher/attendance/sessions/${sessionId}`)
      const second = await need(late, 'POST', '/api/students/attendance/mark', {
        session_id: sessionId,
        code: now.session.code,
      })
      expect(second.status).toBe('late')
      expect(second.message).toContain('late')
      await need(teacher, 'POST', `/api/teacher/attendance/sessions/${sessionId}/close`)
      await signIn(page, { ...ROLES.student, ...CLASS_LATE })
      await visit(page, '/student/attendance?tab=mine')
      const row = page.locator('[data-class-status="late"]').first()
      await expect(row).toBeVisible()
      await expect(row).toContainText('Late')
    } finally {
      if (sessionId) {
        await call(teacher, 'POST', `/api/teacher/attendance/sessions/${sessionId}/close`)
        await call(teacher, 'DELETE', `/api/teacher/attendance/sessions/${sessionId}`)
      }
    }
  })
})
