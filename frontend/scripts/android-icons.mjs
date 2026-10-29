import { mkdir, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'
import { CLEAR, WHITE, mark, onBackground, silhouette } from './logo.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const resDir = join(root, 'android', 'app', 'src', 'main', 'res')

const LAUNCHER = [
  ['mipmap-mdpi', 48],
  ['mipmap-hdpi', 72],
  ['mipmap-xhdpi', 96],
  ['mipmap-xxhdpi', 144],
  ['mipmap-xxxhdpi', 192],
]

const FOREGROUND = [
  ['mipmap-mdpi', 108],
  ['mipmap-hdpi', 162],
  ['mipmap-xhdpi', 216],
  ['mipmap-xxhdpi', 324],
  ['mipmap-xxxhdpi', 432],
]

const SMALL_ICON = [
  ['drawable-mdpi', 24],
  ['drawable-hdpi', 36],
  ['drawable-xhdpi', 48],
  ['drawable-xxhdpi', 72],
  ['drawable-xxxhdpi', 96],
]

const SPLASH_LOGO = [
  ['drawable-mdpi', 96],
  ['drawable-hdpi', 144],
  ['drawable-xhdpi', 192],
  ['drawable-xxhdpi', 288],
  ['drawable-xxxhdpi', 384],
]

const STALE_SPLASH = [
  'drawable',
  'drawable-port-mdpi',
  'drawable-port-hdpi',
  'drawable-port-xhdpi',
  'drawable-port-xxhdpi',
  'drawable-port-xxxhdpi',
  'drawable-land-mdpi',
  'drawable-land-hdpi',
  'drawable-land-xhdpi',
  'drawable-land-xxhdpi',
  'drawable-land-xxxhdpi',
]

const SPLASH_XML = `<?xml version="1.0" encoding="utf-8"?>
<layer-list xmlns:android="http://schemas.android.com/apk/res/android">
    <item android:drawable="@color/splashBackground" />
    <item>
        <bitmap android:src="@drawable/splash_logo" android:gravity="center" />
    </item>
</layer-list>
`

const save = async (folder, name, buffer) => {
  const target = join(resDir, folder)
  await mkdir(target, { recursive: true })
  await writeFile(join(target, name), buffer)
}

const roundIcon = async (size) => {
  const face = await onBackground(size, 0.16, WHITE)
  const circle = Buffer.from(
    `<svg width="${size}" height="${size}"><circle cx="${size / 2}" cy="${size / 2}" r="${size / 2}" fill="#fff"/></svg>`,
  )
  return sharp(face)
    .composite([{ input: circle, blend: 'dest-in' }])
    .png()
    .toBuffer()
}

const run = async () => {
  for (const [folder, size] of LAUNCHER) {
    await save(folder, 'ic_launcher.png', await onBackground(size, 0.14, WHITE))
    await save(folder, 'ic_launcher_round.png', await roundIcon(size))
  }
  for (const [folder, size] of FOREGROUND) {
    await save(folder, 'ic_launcher_foreground.png', await onBackground(size, 0.21, CLEAR))
  }
  for (const [folder, size] of SMALL_ICON) {
    await save(folder, 'ic_stat_notify.png', await silhouette(size))
  }
  for (const [folder, size] of SPLASH_LOGO) {
    await save(folder, 'splash_logo.png', await mark(size))
  }
  await save('drawable-nodpi', 'sust_logo.png', await mark(256))
  for (const folder of STALE_SPLASH) {
    await rm(join(resDir, folder, 'splash.png'), { force: true })
  }
  await save('drawable', 'splash.xml', Buffer.from(SPLASH_XML, 'utf8'))
  console.log('android icons written')
}

run().catch((failure) => {
  console.error(failure)
  process.exitCode = 1
})
