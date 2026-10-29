import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'
import { mark as logoMark } from './logo.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const source = join(root, 'public', 'brand', 'sust-logo.png')

const outDir = join(root, 'public')

const MAROON = '#7A1F2B'

const WHITE = { r: 255, g: 255, b: 255, alpha: 1 }

const CLEAR = { r: 0, g: 0, b: 0, alpha: 0 }

const mark = (size) =>
  sharp(source)
    .trim({ background: WHITE, threshold: 12 })
    .resize(size, size, { fit: 'contain', background: CLEAR })
    .toBuffer()

const onWhite = async (size, padding) => {
  const logo = await mark(Math.round(size * (1 - padding * 2)))
  return sharp({ create: { width: size, height: size, channels: 4, background: WHITE } })
    .composite([{ input: logo, gravity: 'centre' }])
    .png()
    .toBuffer()
}

const maskable = async (size) => {
  const plate = Math.round(size * 0.72)
  const logo = await mark(Math.round(size * 0.5))
  const circle = Buffer.from(
    `<svg width="${plate}" height="${plate}"><circle cx="${plate / 2}" cy="${plate / 2}" r="${plate / 2}" fill="#ffffff"/></svg>`,
  )
  return sharp({
    create: { width: size, height: size, channels: 4, background: MAROON },
  })
    .composite([
      { input: circle, gravity: 'centre' },
      { input: logo, gravity: 'centre' },
    ])
    .png()
    .toBuffer()
}

const badge = async (size) => {
  const inner = Math.round(size * 0.9)
  const silhouette = await sharp(source)
    .trim({ background: WHITE, threshold: 12 })
    .resize(inner, inner, { fit: 'contain', background: WHITE })
    .flatten({ background: WHITE })
    .greyscale()
    .negate()
    .threshold(110)
    .toBuffer()
  const white = await sharp({
    create: { width: inner, height: inner, channels: 3, background: WHITE },
  })
    .joinChannel(silhouette)
    .png()
    .toBuffer()
  return sharp({ create: { width: size, height: size, channels: 4, background: CLEAR } })
    .composite([{ input: white, gravity: 'centre' }])
    .png()
    .toBuffer()
}

const targets = [
  { name: 'icon-192.png', build: () => onWhite(192, 0.08) },
  { name: 'icon-512.png', build: () => onWhite(512, 0.08) },
  { name: 'icon-192-maskable.png', build: () => maskable(192) },
  { name: 'icon-512-maskable.png', build: () => maskable(512) },
  { name: 'apple-touch-icon.png', build: () => onWhite(180, 0.1) },
  { name: 'favicon.png', build: () => onWhite(64, 0.04) },
  { name: 'badge-96.png', build: () => badge(96) },
  { name: 'brand/sust-logo-mark.png', build: () => logoMark(512) },
]

const run = async () => {
  for (const target of targets) {
    const buffer = await target.build()
    await sharp(buffer).toFile(join(outDir, target.name))
    process.stdout.write(`${target.name}\n`)
  }
}

run().catch((failure) => {
  process.stderr.write(`${failure.message}\n`)
  process.exitCode = 1
})
