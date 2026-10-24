import type { FormRules } from 'element-plus'
import type { BirthFieldLabels } from './birth-fields'

/**
 * Shared form rules for birth inputs. FormShell owns validation; tools must
 * pass rules so an empty submit produces field-level feedback instead of a
 * silent no-op.
 */
export function birthFormRules(labels: BirthFieldLabels): FormRules {
  return {
    gender: [{
      required: true,
      message: labels.genderPlaceholder,
      trigger: 'change',
    }],
    date: [{
      required: true,
      message: labels.selectDate,
      trigger: 'change',
    }],
    time: [{
      required: true,
      message: labels.timePlaceholder,
      trigger: 'change',
    }],
    location: [{
      validator: (_rule, value: unknown, callback: (error?: Error) => void) => {
        const location = value as { longitude?: unknown; timezone?: unknown } | null
        if (
          location
          && typeof location.longitude === 'number'
          && Number.isFinite(location.longitude)
          && location.longitude >= -180
          && location.longitude <= 180
          && typeof location.timezone === 'string'
          && location.timezone.trim() !== ''
        ) {
          callback()
          return
        }
        callback(new Error(labels.region.placePlaceholder))
      },
      trigger: 'change',
    }],
  }
}
