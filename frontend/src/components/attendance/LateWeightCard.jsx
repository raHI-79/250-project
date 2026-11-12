import { useEffect, useState } from 'react'
import { Button, Card, CardHeader, Input, useToast } from '../ui'
import api from '../../lib/api'
import { lateWeightProblem } from '../../lib/attendanceTiming'
import { useResource } from '../../lib/hooks'

export default function LateWeightCard({ className }) {
  const toast = useToast()
  const { data, reload } = useResource('/api/admin/attendance/late-weight')
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    if (data && typeof data.weight === 'number') setValue(String(data.weight))
  }, [data])
  const problem = lateWeightProblem(value)
  const save = async () => {
    setSaving(true)
    try {
      await api.put('/api/admin/attendance/late-weight', { weight: Number(value) })
      toast.success('The late weight is saved.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setSaving(false)
    }
  }
  return (
    <Card hover={false} className={className} data-late-weight>
      <CardHeader
        title="How much a late class counts"
        subtitle="Present counts the full class hours, absent counts nothing and late counts this share. With 0.5, two lates equal one class."
      />
      <div className="mt-4 flex flex-wrap items-end gap-3">
        <Input
          label="Late weight"
          inputMode="decimal"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          error={value === '' ? '' : problem}
          hint="From 0 to 1"
          className="w-40"
        />
        <Button loading={saving} disabled={Boolean(problem)} onClick={save}>
          Save
        </Button>
      </div>
    </Card>
  )
}
