type TemporalApi = typeof import('@js-temporal/polyfill').Temporal

let polyfill: Promise<TemporalApi> | null = null

/** Native Temporal first; lazy polyfill only for browsers without it. */
async function temporal(): Promise<TemporalApi> {
  const native = (globalThis as { Temporal?: TemporalApi }).Temporal
  if (native) return native
  if (!polyfill) {
    polyfill = import('@js-temporal/polyfill').then((module) => module.Temporal)
  }
  return polyfill
}

/** RFC 3339 UTC instants for a Gregorian wall time in an IANA time zone. */
export async function wallTimeToInstantCandidates(
  date: string,
  time: string,
  timezone: string,
): Promise<string[]> {
  const [year, month, day] = date.split('-').map(Number)
  const [hour, minute, second = 0] = time.split(':').map(Number)
  const api = await temporal()
  const plain = api.PlainDateTime.from({ year, month, day, hour, minute, second })
  const wall = plain.toString()
  const readings = (['earlier', 'later'] as const).map(disambiguation =>
    api.ZonedDateTime.from(
      { year, month, day, hour, minute, second, timeZone: timezone },
      { disambiguation },
    ),
  )
  return [...new Set(readings
    .filter(zoned => zoned.toPlainDateTime().toString() === wall)
    .map(zoned => zoned.toInstant().toString()))]
    .sort((a, b) => a.localeCompare(b))
}

/** Standard wall-clock + IANA -> UTC. Gaps and folds remain explicit errors. */
export async function wallTimeToInstant(date: string, time: string, timezone: string): Promise<string> {
  const candidates = await wallTimeToInstantCandidates(date, time, timezone)
  if (candidates.length === 0) {
    throw new RangeError(`Wall time ${date}T${time} does not exist in ${timezone}`)
  }
  if (candidates.length > 1) {
    throw new RangeError(`Wall time ${date}T${time} is ambiguous in ${timezone}`)
  }
  return candidates[0]
}
