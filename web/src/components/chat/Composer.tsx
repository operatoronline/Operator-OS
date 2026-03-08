// ============================================================================
// Operator OS — Composer
// Full-featured chat input: auto-resize textarea, file/image upload
// (drag-and-drop + clipboard paste), preview thumbnails, model selector,
// and agent selector. Ports the legacy floating glass treatment.
// ============================================================================

import {
  useRef,
  useState,
  useCallback,
  useEffect,
  type FormEvent,
  type KeyboardEvent,
  type DragEvent,
  type ClipboardEvent,
  type ChangeEvent,
} from 'react'
import {
  PaperPlaneRight,
  Paperclip,
  X,
  Image as ImageIcon,
  File as FileIcon,
  CaretDown,
} from '@phosphor-icons/react'
import { useChatStore } from '../../stores/chatStore'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AttachedFile {
  id: string
  file: File
  previewUrl?: string
  type: 'image' | 'file'
}

interface ComposerProps {
  /** Available models for the model selector */
  models?: string[]
  /** Currently selected model */
  activeModel?: string
  /** Callback when model is changed */
  onModelChange?: (model: string) => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const MAX_FILE_SIZE = 10 * 1024 * 1024 // 10 MB
const MAX_ATTACHMENTS = 5
const ACCEPTED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
const MAX_TEXTAREA_HEIGHT = 160

function generateFileId(): string {
  return `file-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function isImageType(type: string): boolean {
  return ACCEPTED_IMAGE_TYPES.includes(type)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function Composer({ models, activeModel, onModelChange }: ComposerProps) {
  const sendMessage = useChatStore((s) => s.sendMessage)
  const connectionState = useChatStore((s) => s.connectionState)
  const streamingMessageId = useChatStore((s) => s.streamingMessageId)

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [value, setValue] = useState('')
  const [attachments, setAttachments] = useState<AttachedFile[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const [modelOpen, setModelOpen] = useState(false)
  const modelMenuRef = useRef<HTMLDivElement>(null)

  const disabled = connectionState !== 'connected' || !!streamingMessageId
  const canSend = value.trim().length > 0 || attachments.length > 0

  // ─── Auto-resize textarea ───
  const autoResize = useCallback(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`
  }, [])

  useEffect(() => {
    autoResize()
  }, [value, autoResize])

  // ─── Close model dropdown on outside click ───
  useEffect(() => {
    if (!modelOpen) return
    const handler = (e: MouseEvent) => {
      if (modelMenuRef.current && !modelMenuRef.current.contains(e.target as Node)) {
        setModelOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [modelOpen])

  // ─── File processing ───
  const processFiles = useCallback((files: FileList | File[]) => {
    const fileArray = Array.from(files)
    const newAttachments: AttachedFile[] = []

    for (const file of fileArray) {
      if (attachments.length + newAttachments.length >= MAX_ATTACHMENTS) break
      if (file.size > MAX_FILE_SIZE) continue // silently skip oversized

      const isImage = isImageType(file.type)
      const attached: AttachedFile = {
        id: generateFileId(),
        file,
        type: isImage ? 'image' : 'file',
        previewUrl: isImage ? URL.createObjectURL(file) : undefined,
      }
      newAttachments.push(attached)
    }

    if (newAttachments.length > 0) {
      setAttachments((prev) => [...prev, ...newAttachments])
    }
  }, [attachments.length])

  const removeAttachment = useCallback((id: string) => {
    setAttachments((prev) => {
      const removed = prev.find((a) => a.id === id)
      if (removed?.previewUrl) URL.revokeObjectURL(removed.previewUrl)
      return prev.filter((a) => a.id !== id)
    })
  }, [])

  // ─── Cleanup preview URLs on unmount ───
  useEffect(() => {
    return () => {
      for (const a of attachments) {
        if (a.previewUrl) URL.revokeObjectURL(a.previewUrl)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ─── Submit ───
  const handleSubmit = useCallback(
    (e?: FormEvent) => {
      e?.preventDefault()
      if (disabled || !canSend) return

      const text = value.trim()

      // TODO: In C10+ or future tasks, send attachments via WS or upload API
      // For now, we send the text only. Attachments are prepared for the wire format.
      if (text) {
        sendMessage(text)
      }

      // Clear state
      setValue('')
      setAttachments((prev) => {
        for (const a of prev) {
          if (a.previewUrl) URL.revokeObjectURL(a.previewUrl)
        }
        return []
      })

      // Reset textarea height
      requestAnimationFrame(() => {
        if (textareaRef.current) {
          textareaRef.current.style.height = 'auto'
          textareaRef.current.focus()
        }
      })
    },
    [disabled, canSend, value, sendMessage],
  )

  // ─── Keyboard: Enter to send, Shift+Enter for newline ───
  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
        e.preventDefault()
        handleSubmit()
      }
    },
    [handleSubmit],
  )

  // ─── Clipboard paste (images) ───
  const handlePaste = useCallback(
    (e: ClipboardEvent<HTMLTextAreaElement>) => {
      const items = e.clipboardData?.items
      if (!items) return

      const imageFiles: File[] = []
      for (const item of Array.from(items)) {
        if (item.kind === 'file' && isImageType(item.type)) {
          const file = item.getAsFile()
          if (file) imageFiles.push(file)
        }
      }

      if (imageFiles.length > 0) {
        e.preventDefault()
        processFiles(imageFiles)
      }
    },
    [processFiles],
  )

  // ─── Drag and drop ───
  const handleDragEnter = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    // Only unflag if leaving the composer area entirely
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const { clientX, clientY } = e
    if (
      clientX < rect.left ||
      clientX > rect.right ||
      clientY < rect.top ||
      clientY > rect.bottom
    ) {
      setIsDragging(false)
    }
  }, [])

  const handleDragOver = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
  }, [])

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setIsDragging(false)
      if (e.dataTransfer?.files?.length) {
        processFiles(e.dataTransfer.files)
      }
    },
    [processFiles],
  )

  // ─── File input change ───
  const handleFileInputChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      if (e.target.files?.length) {
        processFiles(e.target.files)
        e.target.value = '' // reset so same file can be re-added
      }
    },
    [processFiles],
  )

  // ─── Render ───
  return (
    <div className="max-w-3xl mx-auto w-full">
      <div
        className={`
          glass rounded-[var(--radius)] p-2 transition-all duration-200
          ${isDragging ? 'ring-2 ring-accent ring-offset-2 ring-offset-bg' : ''}
          ${disabled ? 'opacity-60' : ''}
        `}
        onDragEnter={handleDragEnter}
        onDragLeave={handleDragLeave}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        {/* ─── Drag overlay ─── */}
        {isDragging && (
          <div className="flex items-center justify-center gap-2 py-4 text-sm text-accent-text animate-fade-in">
            <ImageIcon size={20} weight="duotone" />
            <span>Drop files here</span>
          </div>
        )}

        {/* ─── Attachment previews ─── */}
        {attachments.length > 0 && !isDragging && (
          <div className="flex gap-2 px-1 pt-1 pb-2 overflow-x-auto">
            {attachments.map((a) => (
              <div
                key={a.id}
                className="relative shrink-0 group animate-scale-in"
              >
                {a.type === 'image' && a.previewUrl ? (
                  <img
                    src={a.previewUrl}
                    alt={a.file.name}
                    className="h-16 w-16 object-cover rounded-lg border border-border-subtle"
                  />
                ) : (
                  <div className="h-16 w-28 flex flex-col items-center justify-center gap-1 rounded-lg border border-border-subtle bg-surface-2 px-2">
                    <FileIcon size={18} className="text-text-dim" />
                    <span className="text-[10px] text-text-secondary truncate w-full text-center">
                      {a.file.name}
                    </span>
                    <span className="text-[9px] text-text-dim">
                      {formatFileSize(a.file.size)}
                    </span>
                  </div>
                )}
                {/* Remove button */}
                <button
                  onClick={() => removeAttachment(a.id)}
                  className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-surface-3 border border-border
                    flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity
                    hover:bg-error hover:text-white hover:border-error cursor-pointer"
                  aria-label={`Remove ${a.file.name}`}
                >
                  <X size={10} weight="bold" />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* ─── Input row ─── */}
        {!isDragging && (
          <div className="flex items-end gap-2">
            {/* Attach button — 44px touch target on mobile */}
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled || attachments.length >= MAX_ATTACHMENTS}
              className="shrink-0 w-11 h-11 md:w-9 md:h-9 flex items-center justify-center rounded-[10px]
                text-text-dim hover:text-text hover:bg-surface-2/60
                active:scale-95 active:opacity-80
                transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
              aria-label="Attach file"
            >
              <Paperclip size={18} />
            </button>

            {/* Hidden file input */}
            <input
              ref={fileInputRef}
              type="file"
              multiple
              accept="image/*,.pdf,.txt,.md,.json,.csv,.xml,.yaml,.yml,.log"
              className="hidden"
              onChange={handleFileInputChange}
            />

            {/* Textarea — 16px on mobile to prevent iOS Safari zoom */}
            <textarea
              ref={textareaRef}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={handleKeyDown}
              onPaste={handlePaste}
              rows={1}
              disabled={disabled}
              placeholder={
                streamingMessageId
                  ? 'Waiting for response…'
                  : connectionState !== 'connected'
                    ? 'Connecting…'
                    : 'Message Operator OS…'
              }
              className="flex-1 resize-none bg-transparent text-[var(--text)]
                text-[16px] md:text-[15px] leading-[1.4]
                border-none outline-none py-2 px-2.5
                placeholder:text-[var(--text-dim)]
                disabled:opacity-50
                font-[family-name:var(--font)]"
              style={{ maxHeight: MAX_TEXTAREA_HEIGHT }}
              aria-label="Chat message input"
            />

            {/* Model selector (when models are available) */}
            {models && models.length > 0 && (
              <div className="relative shrink-0" ref={modelMenuRef}>
                <button
                  type="button"
                  onClick={() => setModelOpen(!modelOpen)}
                  disabled={disabled}
                  className="flex items-center gap-1 h-9 px-2.5 rounded-[10px]
                    text-[11px] font-medium text-text-dim
                    hover:text-text-secondary hover:bg-surface-2/60
                    transition-colors disabled:opacity-30 cursor-pointer"
                  aria-label="Select model"
                >
                  <span className="max-w-[80px] truncate">
                    {activeModel || models[0]}
                  </span>
                  <CaretDown size={10} weight="bold" />
                </button>

                {modelOpen && (
                  <div
                    className="absolute bottom-full right-0 mb-2 w-48
                      glass rounded-[var(--radius-md)] py-1
                      animate-fade-slide-down z-50"
                  >
                    {models.map((model) => (
                      <button
                        key={model}
                        onClick={() => {
                          onModelChange?.(model)
                          setModelOpen(false)
                        }}
                        className={`
                          w-full text-left px-3 py-2 text-xs font-medium
                          transition-colors cursor-pointer
                          ${model === (activeModel || models[0])
                            ? 'text-accent-text bg-accent-subtle/50'
                            : 'text-text-secondary hover:text-text hover:bg-surface-2/60'
                          }
                        `}
                      >
                        {model}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Send button — 44px touch target on mobile */}
            <button
              type="button"
              onClick={() => handleSubmit()}
              disabled={disabled || !canSend}
              className="shrink-0 w-11 h-11 md:w-9 md:h-9 flex items-center justify-center rounded-[10px]
                bg-accent text-white
                hover:opacity-85 active:scale-[0.94]
                transition-all duration-150
                disabled:opacity-25 disabled:cursor-not-allowed cursor-pointer"
              aria-label="Send message"
            >
              <PaperPlaneRight size={16} weight="fill" />
            </button>
          </div>
        )}
      </div>

      {/* ─── Hint text ─── */}
      <div className="flex items-center justify-between mt-1.5 px-2">
        <span className="text-[10px] text-text-dim">
          {attachments.length > 0
            ? `${attachments.length}/${MAX_ATTACHMENTS} files`
            : 'Enter to send · Shift+Enter for newline'}
        </span>
        {attachments.length > 0 && (
          <button
            onClick={() => {
              for (const a of attachments) {
                if (a.previewUrl) URL.revokeObjectURL(a.previewUrl)
              }
              setAttachments([])
            }}
            className="text-[10px] text-text-dim hover:text-error transition-colors cursor-pointer"
          >
            Clear all
          </button>
        )}
      </div>
    </div>
  )
}
