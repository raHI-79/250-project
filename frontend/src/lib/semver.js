const CODE_STEP = 100

const PARTS = 3

export const versionCode = (version) => {
  const numbers = String(version || '')
    .trim()
    .replace(/^v/i, '')
    .split('-')[0]
    .split('.')
  let code = 0
  for (let index = 0; index < PARTS; index += 1) {
    const piece = Number.parseInt(numbers[index] || '0', 10)
    code = code * CODE_STEP + (Number.isFinite(piece) && piece > 0 ? piece : 0)
  }
  return code
}

export const newerThan = (latest, installed) => versionCode(latest) > versionCode(installed)
