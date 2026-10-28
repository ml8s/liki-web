type TemporalApi = typeof import('@js-temporal/polyfill').Temporal

/** Canonical Engine time input. Calendar dates are projections, never inputs. */
export interface ChartInput {
  standard_time: string
  longitude: number
}

/** The product calendar defaults to China Standard Time for date-only inputs. */
export const CALENDAR_STANDARD_TIME_ZONE = 'Asia/Shanghai'
const CALENDAR_STANDARD_OFFSET = '+08:00'

interface ZonedDateTimeParts {
  year: number
  month: number
  day: number
  hour: number
  minute: number
  second: number
}

function zonedParts(date: Date, timezone: string): ZonedDateTimeParts {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(date)
  const value = (type: Intl.DateTimeFormatPartTypes): number => {
    const item = parts.find(part => part.type === type)
    if (!item) throw new RangeError(`Time zone ${timezone} produced incomplete time parts`)
    return Number.parseInt(item.value, 10)
  }
  return {
    year: value('year'),
    month: value('month'),
    day: value('day'),
    hour: value('hour') % 24,
    minute: value('minute'),
    second: value('second'),
  }
}

function parseOffsetMinutes(value: string): number {
  const match = /^GMT([+-])(\d{2}):(\d{2})$/.exec(value)
  if (!match) return 0
  const sign = match[1] === '-' ? -1 : 1
  return sign * (Number(match[2]) * 60 + Number(match[3]))
}

function offsetMinutesAt(date: Date, timezone: string): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: timezone,
    timeZoneName: 'longOffset',
  }).formatToParts(date)
  const offset = parts.find(part => part.type === 'timeZoneName')?.value ?? 'GMT'
  return parseOffsetMinutes(offset)
}

/** Choose the numerically smaller offset; common DST shifts clocks forward. */
function standardOffsetMinutesAt(date: Date, timezone: string): number {
  const { year } = zonedParts(date, timezone)
  const samples = [
    new Date(Date.UTC(year, 0, 1, 12)),
    new Date(Date.UTC(year, 6, 1, 12)),
  ]
  return samples
    .map(sample => offsetMinutesAt(sample, timezone))
    .reduce((standard, offset) => (
      offset < standard ? offset : standard
    ), Number.POSITIVE_INFINITY)
}

function formatOffset(minutes: number): string {
  if (!Number.isInteger(minutes)) throw new RangeError('Time zone offset must resolve to whole minutes')
  const sign = minutes < 0 ? '-' : '+'
  const absolute = Math.abs(minutes)
  const hours = String(Math.floor(absolute / 60)).padStart(2, '0')
  const mins = String(absolute % 60).padStart(2, '0')
  return `${sign}${hours}:${mins}`
}

function formatZonedStandardTime(date: Date, timezone: string): string {
  const standardOffset = standardOffsetMinutesAt(date, timezone)
  const standardWall = new Date(date.getTime() + standardOffset * 60_000)
  const parts = zonedParts(standardWall, 'UTC')
  const pad = (value: number): string => String(value).padStart(2, '0')
  return [
    `${parts.year}-${pad(parts.month)}-${pad(parts.day)}`,
    `T${pad(parts.hour)}:${pad(parts.minute)}:${pad(parts.second)}`,
    formatOffset(standardOffset),
  ].join('')
}

/** UTC instant + IANA zone -> Engine standard_time (never a DST offset). */
export function standardTimeFromInstant(instant: string, timezone: string): string {
  const date = new Date(instant)
  if (Number.isNaN(date.getTime())) {
    throw new RangeError(`Invalid instant ${instant}`)
  }
  return formatZonedStandardTime(date, timezone)
}

/** Date-only product input -> a stable midday standard_time. */
export function standardTimeForDate(
  date: string,
  timezone: string = 'Asia/Shanghai',
  time = '12:00',
): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) {
    throw new RangeError(`Invalid calendar date ${date}`)
  }
  if (!/^\d{2}:\d{2}$/.test(time)) {
    throw new RangeError(`Invalid calendar time ${time}`)
  }
  const offset = timezone === CALENDAR_STANDARD_TIME_ZONE
    ? CALENDAR_STANDARD_OFFSET
    : formatOffset(standardOffsetMinutesAt(new Date(`${date}T${time}:00Z`), timezone))
  return `${date}T${time}:00${offset}`
}

/** Civil wall time + IANA zone -> Engine standard_time. */
export function wallTimeToStandardTime(
  date: string,
  time: string,
  timezone: string,
): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || !/^\d{2}:\d{2}$/.test(time)) {
    throw new RangeError(`Invalid wall time ${date}T${time}`)
  }
  const offset = standardOffsetMinutesAt(new Date(`${date}T${time}:00Z`), timezone)
  return `${date}T${time}:00${formatOffset(offset)}`
}

/** Year-number selector -> a mid-year representative standard_time. */
export function yearToStandardTime(
  year: number,
  timezone: string = 'Asia/Shanghai',
): string {
  if (!Number.isInteger(year) || year < 1) {
    throw new RangeError(`Invalid selector year ${year}`)
  }
  return standardTimeForDate(`${String(year).padStart(4, '0')}-06-15`, timezone)
}

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

/** RFC 3339 instants for a Gregorian wall time in an IANA time zone. */
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

/** Unambiguous wall-clock + IANA -> instant. Gaps and folds remain explicit errors. */
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

/** Display-only zoned formatting; callers never do timezone arithmetic. */
export function formatInstant(
  instant: string,
  timezone: string,
  locale?: string,
  options?: Intl.DateTimeFormatOptions,
): string {
  return new Intl.DateTimeFormat(locale, {
    timeZone: timezone,
    dateStyle: 'medium',
    timeStyle: 'short',
    ...options,
  }).format(new Date(instant))
}
