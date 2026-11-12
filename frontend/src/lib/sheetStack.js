const open = []

export const pushSheet = (entry) => {
  open.push(entry)
  return () => {
    const index = open.indexOf(entry)
    if (index >= 0) open.splice(index, 1)
  }
}

export const openSheets = () => open.length

export const closeTopSheet = () => {
  const entry = open[open.length - 1]
  if (!entry) return false
  entry.close?.()
  return true
}
