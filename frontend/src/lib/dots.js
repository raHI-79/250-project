export const dots = (...parts) =>
  parts
    .flat()
    .map((part) => (part === null || part === undefined ? '' : String(part).trim()))
    .filter(Boolean)
    .join(' · ')

export default dots
