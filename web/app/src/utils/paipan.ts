// 共享：出生信息（档案或手填）→ bazi.chart 最小命盘。
import {
  engineBaziChart,
  engineCityCoords,
  engineSolarTime,
  type BaziChartData,
} from '@/api/engine'

export interface BirthInfo {
  gender: string
  calendar?: string
  birth_date: string
  birth_time: string
  birth_time_known?: string
  birth_place?: string
}

export async function baziChartFor(birth: BirthInfo): Promise<BaziChartData> {
  if (birth.calendar === 'lunar') throw new Error('免费预览暂支持公历出生时间')
  const known = birth.birth_time_known ?? 'known'
  if (known === 'unknown' || !birth.birth_time) throw new Error('需要出生时间')
  let longitude = 120
  if (known === 'known' && birth.birth_place) {
    let coords: { longitude?: number } | null = null
    try {
      coords = await engineCityCoords(birth.birth_place)
    } catch {
      throw new Error(`无法识别出生地「${birth.birth_place}」，无法做真太阳时修正；请改用常见城市名，或移除此出生地后重试`)
    }
    if (!coords || typeof coords.longitude !== 'number') {
      throw new Error(`未找到出生地「${birth.birth_place}」；请改用常见城市名，或移除此出生地后重试`)
    }
    longitude = coords.longitude
  }
  const datetime = `${birth.birth_date}T${birth.birth_time}:00+08:00`
  const t = await engineSolarTime(datetime, longitude)
  const solar = known === 'known' && birth.birth_place ? t.solar : datetime
  return engineBaziChart(solar, birth.gender)
}