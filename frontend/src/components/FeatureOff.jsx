import { Link } from 'react-router-dom'
import { PowerOff } from 'lucide-react'
import { Button, EmptyState } from './ui'
import { featureOffMessage } from '../lib/features'
import paths from '../lib/paths'

export default function FeatureOff({ feature, compact = false }) {
  return (
    <div data-feature-off={feature} className={compact ? '' : 'pt-6'}>
      <EmptyState
        icon={PowerOff}
        title={featureOffMessage(feature)}
        text="It will be back here as soon as the department turns it on again."
        action={
          compact ? null : (
            <Button as={Link} to={paths.root()} variant="secondary" size="sm">
              Go home
            </Button>
          )
        }
      />
    </div>
  )
}
