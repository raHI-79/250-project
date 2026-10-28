const BLOBS = [
  { tone: 'brand', className: 'top-[-14%] left-[-18%] size-[28rem] sm:size-[36rem]', delay: '0s' },
  { tone: 'gold', className: 'top-[18%] right-[-22%] size-[24rem] sm:size-[32rem]', delay: '-9s' },
  { tone: 'brand', className: 'bottom-[-20%] left-[12%] size-[26rem] sm:size-[34rem]', delay: '-17s' },
]

const FILL = {
  brand: 'var(--c-blob-brand)',
  gold: 'var(--c-blob-gold)',
}

export default function GlassBackdrop() {
  return (
    <div aria-hidden="true" className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      <div className="absolute inset-0 bg-page" />
      {BLOBS.map((blob, index) => (
        <span
          key={index}
          style={{ backgroundColor: FILL[blob.tone], animationDelay: blob.delay }}
          className={`animate-blob absolute rounded-full blur-3xl ${blob.className}`}
        />
      ))}
    </div>
  )
}
