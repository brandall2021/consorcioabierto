import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'

export function Modal({
  title,
  open,
  onClose,
  children,
}: {
  title: string
  open: boolean
  onClose: () => void
  children: ReactNode
}) {
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) {
      dialog.showModal()
    } else if (!open && dialog.open) {
      dialog.close()
    }
  }, [open])

  return (
    <dialog
      ref={ref}
      onClose={onClose}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      className="m-auto w-full max-w-lg rounded-lg border bg-white p-0 shadow-xl backdrop:bg-black/30"
      aria-labelledby="modal-title"
    >
      <div className="flex items-center justify-between border-b px-4 py-3">
        <h2 id="modal-title" className="text-base font-medium">
          {title}
        </h2>
        <button
          type="button"
          onClick={onClose}
          aria-label="Cerrar"
          className="rounded-md px-2 py-1 text-gray-500 hover:bg-gray-100"
        >
          ✕
        </button>
      </div>
      <div className="p-4">{children}</div>
    </dialog>
  )
}