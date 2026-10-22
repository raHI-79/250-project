export const SUST_LOGO = '/brand/sust-logo.png'

export const CSE_LOGO = '/brand/cse-logo.png'

export const applyFavicon = () => {
  const image = new Image()
  image.onload = () => {
    const link = document.querySelector('link[rel="icon"]')
    if (link) link.setAttribute('href', CSE_LOGO)
  }
  image.src = CSE_LOGO
}
