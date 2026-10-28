import { useCallback, useState } from 'react'
import Cropper from 'react-easy-crop'
import { Button, Sheet } from '../ui'
import { cropToSquare } from '../../lib/crop'

export default function CropDialog({ open, source, fileName = 'photo.jpg', onCancel, onDone }) {
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [area, setArea] = useState(null)
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState('')
  const onCropComplete = useCallback((_, pixels) => setArea(pixels), [])
  const confirm = async () => {
    if (!area) return
    setBusy(true)
    setProblem('')
    try {
      onDone(await cropToSquare(source, area, fileName))
    } catch (failure) {
      setProblem(failure.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Sheet
      open={open}
      onClose={onCancel}
      title="Crop your photo"
      className="max-w-md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={confirm} loading={busy} disabled={!area}>
            Use this photo
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="relative h-64 w-full overflow-hidden rounded-xl bg-night sm:h-72">
          {source && (
            <Cropper
              image={source}
              crop={crop}
              zoom={zoom}
              aspect={1}
              cropShape="round"
              showGrid={false}
              onCropChange={setCrop}
              onZoomChange={setZoom}
              onCropComplete={onCropComplete}
            />
          )}
        </div>
        <label className="flex items-center gap-3 text-sm text-muted">
          Zoom
          <input
            type="range"
            min={1}
            max={3}
            step={0.01}
            value={zoom}
            onChange={(event) => setZoom(Number(event.target.value))}
            className="h-1.5 min-w-0 flex-1 accent-brand-600"
          />
        </label>
        {problem && <p className="text-sm text-danger-600">{problem}</p>}
      </div>
    </Sheet>
  )
}
