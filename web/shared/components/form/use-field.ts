import { computed, toValue, useId, type MaybeRefOrGetter } from 'vue'

/** Shared field ID and ARIA description contract for all controls. */
export function useFieldA11y(
  hint: MaybeRefOrGetter<string | undefined>,
  error: MaybeRefOrGetter<string | undefined>,
) {
  const fieldId = useId()
  const describedBy = computed(() => {
    if (toValue(error)) return `${fieldId}-error`
    return toValue(hint) ? `${fieldId}-hint` : undefined
  })
  const invalid = computed(() => Boolean(toValue(error)))

  return { fieldId, describedBy, invalid }
}
