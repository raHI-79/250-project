const OUTPUT_SIZE = 512

const OUTPUT_TYPE = 'image/jpeg'

const OUTPUT_QUALITY = 0.92

const loadImage = (source) =>
  new Promise((resolve, reject) => {
    const image = new Image()
    image.addEventListener('load', () => resolve(image))
    image.addEventListener('error', () => reject(new Error('The image could not be opened.')))
    image.src = source
  })

export const cropToSquare = async (source, area, name) => {
  const image = await loadImage(source)
  const canvas = document.createElement('canvas')
  canvas.width = OUTPUT_SIZE
  canvas.height = OUTPUT_SIZE
  const context = canvas.getContext('2d')
  context.imageSmoothingQuality = 'high'
  context.drawImage(
    image,
    area.x,
    area.y,
    area.width,
    area.height,
    0,
    0,
    OUTPUT_SIZE,
    OUTPUT_SIZE,
  )
  const blob = await new Promise((resolve) => canvas.toBlob(resolve, OUTPUT_TYPE, OUTPUT_QUALITY))
  if (!blob) throw new Error('The image could not be prepared.')
  return new File([blob], name, { type: OUTPUT_TYPE })
}

