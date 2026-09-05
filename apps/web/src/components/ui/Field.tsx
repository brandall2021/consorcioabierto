import { useId } from 'react'
import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from 'react'

interface FieldProps {
  label: string
  error?: string | null
  hint?: string
  required?: boolean
  children: (id: string) => ReactNode
}

export function Field({ label, error, hint, required, children }: FieldProps) {
  const id = useId()
  return (
    <div>
      <label htmlFor={id} className="text-sm text-gray-600">
        {label}
        {required && <span className="text-red-600"> *</span>}
      </label>
      {children(id)}
      {hint && !error && (
        <p id={`${id}-hint`} className="mt-1 text-xs text-gray-400">
          {hint}
        </p>
      )}
      {error && (
        <p id={`${id}-error`} className="mt-1 text-xs text-red-600" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      id={props.id}
      className={`mt-1 w-full rounded-md border px-3 py-1.5 text-sm disabled:opacity-40 ${props.className ?? ''}`}
    />
  )
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      id={props.id}
      className={`mt-1 w-full rounded-md border px-3 py-1.5 text-sm disabled:opacity-40 ${props.className ?? ''}`}
    >
      {props.children}
    </select>
  )
}