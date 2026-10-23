export type FieldSize = 'sm' | 'md' | 'lg'

export interface BaseFieldProps {
  name: string
  label: string
  prop?: string
  hint?: string
  error?: string
  required?: boolean
  disabled?: boolean
  readonly?: boolean
  placeholder?: string
  size?: FieldSize
}

export interface SelectOption<T extends string | number> {
  value: T
  label: string
  disabled?: boolean
}

export interface ChoiceOption<T extends string | number> {
  value: T
  label: string
  description?: string
  disabled?: boolean
}
