import { access, copyFile, mkdir } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const source = join(root, 'android-config', 'google-services.json')

const target = join(root, 'android', 'app', 'google-services.json')

const run = async () => {
  try {
    await access(source)
  } catch {
    console.log('android-config/google-services.json is missing, push will not work in the app')
    return
  }
  await mkdir(dirname(target), { recursive: true })
  await copyFile(source, target)
  console.log('copied google-services.json into the Android project')
}

run().catch((failure) => {
  console.error(failure)
  process.exitCode = 1
})
