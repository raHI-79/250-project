const BATCH_NAME = /^[A-Z]{2,6}-[0-9]{2}$/

export const cleanBatchName = (value) => String(value || '').trim().toUpperCase()

export const batchNameProblem = (value) =>
  BATCH_NAME.test(cleanBatchName(value)) ? '' : 'Batch name must look like CSE-25.'

export const REG_NO_MIN = 6

export const REG_NO_MAX = 20

const REG_NO = /^20[0-9]{4,18}$/

export const cleanRegNo = (value) => String(value || '').trim()

export const regNoProblem = (value) => {
  const regNo = cleanRegNo(value)
  if (!regNo) return 'Registration number is required.'
  if (!/^[0-9]+$/.test(regNo)) return 'Registration number may only use digits.'
  if (!REG_NO.test(regNo))
    return `Registration number must start with 20 and be ${REG_NO_MIN} to ${REG_NO_MAX} digits long.`
  return ''
}

export const batchFromRegNo = (value) => {
  const regNo = cleanRegNo(value)
  return /^20[0-9]{2}/.test(regNo) ? `CSE-${regNo.slice(2, 4)}` : ''
}

export const batchClosedText = (batch) =>
  `${batch} is not open for registration. Please contact the department office.`
