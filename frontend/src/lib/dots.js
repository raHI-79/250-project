export const dots = (...parts) =>
  parts
    .flat(Infinity)
    .map((part) => (part === null || part === undefined ? '' : String(part).trim()))
    .filter(Boolean)
    .join(' · ')

export default dots
