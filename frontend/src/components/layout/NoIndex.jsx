import { useEffect } from 'react'

export default function NoIndex() {
  useEffect(() => {
    const tag = document.createElement('meta')
    tag.name = 'robots'
    tag.content = 'noindex,nofollow'
    document.head.appendChild(tag)
    return () => tag.remove()
  }, [])
  return null
}
