export type AppExperience = 'classic' | 'consultation'

const PREFERENCE_KEY = 'liki.preferred-app-experience'

/** Reads the user's saved template choice; null means "use the build default". */
export function readPreferredAppExperience(): AppExperience | null {
  try {
    const value = localStorage.getItem(PREFERENCE_KEY)
    return value === 'classic' || value === 'consultation' ? value : null
  } catch {
    return null
  }
}

/** Persists the template choice; the next full load resolves `/app` with it. */
export function writePreferredAppExperience(experience: AppExperience): void {
  try {
    localStorage.setItem(PREFERENCE_KEY, experience)
  } catch {
    // Storage may be unavailable (private mode); the build default still applies.
  }
}
