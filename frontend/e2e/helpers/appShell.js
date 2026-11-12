import { readFileSync } from 'node:fs'

const APP_API = 'https://sustclassexamcoordinationsystem.onrender.com'

const APP_VERSION = JSON.parse(readFileSync(new URL('../../package.json', import.meta.url), 'utf8')).version

const LOCAL_API = process.env.E2E_API_URL || 'http://localhost:8080'

const PLUGINS = {
  App: ['getInfo', 'exitApp', 'getState', 'minimizeApp'],
  Browser: ['open', 'close'],
  Filesystem: ['downloadFile', 'deleteFile', 'getUri', 'stat'],
  PushNotifications: [
    'checkPermissions',
    'requestPermissions',
    'register',
    'unregister',
    'removeAllListeners',
    'getDeliveredNotifications',
  ],
  StatusBar: ['setStyle', 'setBackgroundColor', 'setOverlaysWebView', 'show', 'hide'],
  ApkInstaller: ['canInstall', 'install', 'openSettings'],
  AttendanceBeacon: [
    'status',
    'ensure',
    'enableBluetooth',
    'start',
    'update',
    'stop',
    'ensureLocation',
    'openSettings',
    'openLocationSettings',
  ],
  AttendanceMark: ['deviceId', 'status', 'ensure', 'enableBluetooth', 'scan', 'position', 'confirm'],
}

function installBridge({ plugins, version }) {
  const ready = {
    supported: true,
    bluetooth_on: true,
    nearby: true,
    place: true,
    location_on: true,
    advertising: false,
  }
  const answers = {
    'App.getInfo': { name: 'SUST CSE Portal', id: 'edu.sust.cse.portal', version, build: '1' },
    'App.getState': { isActive: true },
    'PushNotifications.checkPermissions': { receive: 'denied' },
    'PushNotifications.requestPermissions': { receive: 'denied' },
    'PushNotifications.getDeliveredNotifications': { notifications: [] },
    'ApkInstaller.canInstall': { allowed: false },
    'Filesystem.downloadFile': { path: '' },
    'AttendanceBeacon.status': ready,
    'AttendanceBeacon.ensure': ready,
    'AttendanceBeacon.ensureLocation': ready,
    'AttendanceMark.deviceId': { device_id: 'e2e-phone' },
    'AttendanceMark.status': ready,
    'AttendanceMark.ensure': ready,
    'AttendanceMark.scan': { token: '' },
    'AttendanceMark.position': { found: false },
    'AttendanceMark.confirm': { available: false, confirmed: false },
  }
  let next = 0
  window.androidBridge = { postMessage() {} }
  window.Capacitor = {
    PluginHeaders: Object.entries(plugins).map(([name, methods]) => ({
      name,
      methods: [
        ...methods.map((method) => ({ name: method, rtype: 'promise' })),
        { name: 'addListener', rtype: 'callback' },
        { name: 'removeListener', rtype: 'promise' },
      ],
    })),
    nativePromise: (plugin, method) => Promise.resolve(answers[`${plugin}.${method}`] || {}),
    nativeCallback: () => {
      next += 1
      return `e2e-${next}`
    },
  }
}

export const localURL = (address) =>
  address.startsWith(APP_API) ? address.replace(APP_API, LOCAL_API) : address

export async function asApp(page) {
  await page.addInitScript(installBridge, { plugins: PLUGINS, version: APP_VERSION })
  await page.route(`${APP_API}/**`, async (route) => {
    const target = route.request().url().replace(APP_API, LOCAL_API)
    try {
      const response = await route.fetch({ url: target })
      await route.fulfill({ response })
    } catch {
      await route.abort().catch(() => {})
    }
  })
}

export const appCanReach = (key) => key === 'student' || key === 'cr' || key === 'teacher'
