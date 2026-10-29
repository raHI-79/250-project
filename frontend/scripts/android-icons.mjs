import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const source = join(root, 'public', 'brand', 'sust-logo.png')

const resDir = join(root, 'android', 'app', 'src', 'main', 'res')

const MAROON = { r: 122, g: 31, b: 43, alpha: 1 }

const WHITE = { r: 255, g: 255, b: 255, alpha: 1 }

const CLEAR = { r: 0, g: 0, b: 0, alpha: 0 }

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

const SPLASH = [
  ['drawable-port-mdpi', 320, 480],
  ['drawable-port-hdpi', 480, 800],
  ['drawable-port-xhdpi', 720, 1280],
  ['drawable-port-xxhdpi', 960, 1600],
  ['drawable-port-xxxhdpi', 1280, 1920],
  ['drawable-land-mdpi', 480, 320],
  ['drawable-land-hdpi', 800, 480],
  ['drawable-land-xhdpi', 1280, 720],
  ['drawable-land-xxhdpi', 1600, 960],
  ['drawable-land-xxxhdpi', 1920, 1280],
  ['drawable', 480, 800],
]

const mark = (size) =>
  sharp(source)
    .trim({ background: WHITE, threshold: 12 })
    .resize(size, size, { fit: 'contain', background: CLEAR })
    .toBuffer()

const save = async (folder, name, buffer) => {
  const target = join(resDir, folder)
  await mkdir(target, { recursive: true })
  await writeFile(join(target, name), buffer)
}

const square = async (size, padding, background) => {
  const logo = await mark(Math.round(size * (1 - padding * 2)))
  return sharp({ create: { width: size, height: size, channels: 4, background } })
    .composite([{ input: logo, gravity: 'centre' }])
    .png()
    .toBuffer()
}

const roundIcon = async (size) => {
  const face = await square(size, 0.16, WHITE)
  const circle = Buffer.from(
    `<svg width="${size}" height="${size}"><circle cx="${size / 2}" cy="${size / 2}" r="${size / 2}" fill="#fff"/></svg>`,
  )
  return sharp(face)
    .composite([{ input: circle, blend: 'dest-in' }])
    .png()
    .toBuffer()
}

const monochrome = async (size) => {
  const logo = await mark(Math.round(size * 0.82))
  return sharp({ create: { width: size, height: size, channels: 4, background: CLEAR } })
    .composite([{ input: logo, gravity: 'centre' }])
    .ensureAlpha()
    .toColourspace('b-w')
    .png()
    .toBuffer()
}

const splash = async (width, height) => {
  const side = Math.round(Math.min(width, height) * 0.3)
  const logo = await square(side, 0.12, WHITE)
  const rounded = await sharp(logo)
    .composite([
      {
        input: Buffer.from(
          `<svg width="${side}" height="${side}"><rect width="${side}" height="${side}" rx="${Math.round(side * 0.22)}" fill="#fff"/></svg>`,
        ),
        blend: 'dest-in',
      },
    ])
    .png()
    .toBuffer()
  return sharp({ create: { width, height, channels: 4, background: MAROON } })
    .composite([{ input: rounded, gravity: 'centre' }])
    .png()
    .toBuffer()
}

const run = async () => {
  for (const [folder, size] of LAUNCHER) {
    await save(folder, 'ic_launcher.png', await square(size, 0.14, WHITE))
    await save(folder, 'ic_launcher_round.png', await roundIcon(size))
  }
  for (const [folder, size] of FOREGROUND) {
    await save(folder, 'ic_launcher_foreground.png', await square(size, 0.28, CLEAR))
  }
  for (const [folder, size] of SMALL_ICON) {
    await save(folder, 'ic_stat_notify.png', await monochrome(size))
  }
  for (const [folder, width, height] of SPLASH) {
    await save(folder, 'splash.png', await splash(width, height))
  }
  console.log('android icons written')
}

run().catch((failure) => {
  console.error(failure)
  process.exitCode = 1
})
