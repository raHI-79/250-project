import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

export const SOURCE = join(root, 'public', 'brand', 'sust-logo.png')

export const WHITE = { r: 255, g: 255, b: 255, alpha: 1 }

export const CLEAR = { r: 0, g: 0, b: 0, alpha: 0 }

const NEAR_WHITE = 240

const nearWhite = (data, index) =>
  data[index] >= NEAR_WHITE && data[index + 1] >= NEAR_WHITE && data[index + 2] >= NEAR_WHITE

const clearOutside = (data, width, height) => {
  const seen = new Uint8Array(width * height)
  const queue = []
  const push = (x, y) => {
    if (x < 0 || y < 0 || x >= width || y >= height) return
    const cell = y * width + x
    if (seen[cell]) return
    if (!nearWhite(data, cell * 4)) return
    seen[cell] = 1
    queue.push(cell)
  }
  for (let x = 0; x < width; x += 1) {
    push(x, 0)
    push(x, height - 1)
  }
  for (let y = 0; y < height; y += 1) {
    push(0, y)
    push(width - 1, y)
  }
  while (queue.length > 0) {
    const cell = queue.pop()
    data[cell * 4 + 3] = 0
    const x = cell % width
    const y = (cell - x) / width
    push(x - 1, y)
    push(x + 1, y)
    push(x, y - 1)
    push(x, y + 1)
  }
}

export const cutout = async () => {
  const { data, info } = await sharp(SOURCE)
    .trim({ background: WHITE, threshold: 12 })
    .ensureAlpha()
    .raw()
    .toBuffer({ resolveWithObject: true })
  clearOutside(data, info.width, info.height)
  return sharp(data, { raw: { width: info.width, height: info.height, channels: 4 } })
    .png()
    .toBuffer()
}

export const mark = async (size) =>
  sharp(await cutout())
    .resize(size, size, { fit: 'contain', background: CLEAR })
    .png()
    .toBuffer()

export const silhouette = async (size) => {
  const { data, info } = await sharp(await mark(size))
    .ensureAlpha()
    .raw()
    .toBuffer({ resolveWithObject: true })
  for (let index = 0; index < data.length; index += 4) {
    data[index] = 255
    data[index + 1] = 255
    data[index + 2] = 255
  }
  return sharp(data, { raw: { width: info.width, height: info.height, channels: 4 } })
    .png()
    .toBuffer()
}

export const onBackground = async (size, padding, background) => {
  const logo = await mark(Math.round(size * (1 - padding * 2)))
  return sharp({ create: { width: size, height: size, channels: 4, background } })
    .composite([{ input: logo, gravity: 'centre' }])
    .png()
    .toBuffer()
}
