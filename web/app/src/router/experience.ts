export const appExperiences = ['classic', 'consultation'] as const

export type AppExperience = (typeof appExperiences)[number]
