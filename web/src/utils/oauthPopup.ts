// ============================================================================
// Operator OS — OAuth Popup Utility
// Opens OAuth authorization in a centered popup window and listens for the
// callback result via postMessage or polling.
// ============================================================================

export interface OAuthPopupResult {
  success: boolean
  integration_id?: string
  provider?: string
  error?: string
  code?: string
}

interface PopupOptions {
  url: string
  title?: string
  width?: number
  height?: number
}

/** Calculate centered popup position */
function getCenteredPosition(width: number, height: number) {
  const left = Math.max(0, Math.round(window.screenX + (window.outerWidth - width) / 2))
  const top = Math.max(0, Math.round(window.screenY + (window.outerHeight - height) / 2))
  return { left, top }
}

/**
 * Open an OAuth authorization URL in a popup window.
 * Resolves when the popup posts a message back or navigates to the callback URL.
 * Rejects if the popup is blocked or closed by the user.
 */
export function openOAuthPopup(options: PopupOptions): Promise<OAuthPopupResult> {
  const { url, title = 'Connect Integration', width = 520, height = 700 } = options
  const { left, top } = getCenteredPosition(width, height)

  const features = [
    `width=${width}`,
    `height=${height}`,
    `left=${left}`,
    `top=${top}`,
    'menubar=no',
    'toolbar=no',
    'location=yes',   // Show URL bar for security
    'status=no',
    'resizable=yes',
    'scrollbars=yes',
  ].join(',')

  return new Promise((resolve, reject) => {
    const popup = window.open(url, title, features)

    if (!popup || popup.closed) {
      reject(new Error('Popup was blocked by the browser. Please allow popups for this site.'))
      return
    }

    // Focus the popup
    popup.focus()

    let resolved = false
    const popupRef = popup

    // ─── Listen for postMessage from callback page ───
    function onMessage(event: MessageEvent) {
      // Only accept messages from our own origin
      if (event.origin !== window.location.origin) return
      if (!event.data || event.data.type !== 'os:oauth:callback') return

      resolved = true
      cleanup()
      popupRef.close()
      resolve(event.data.result as OAuthPopupResult)
    }

    // ─── Poll for popup closed (user dismissed) ───
    const pollInterval = setInterval(() => {
      if (popupRef.closed) {
        if (!resolved) {
          resolved = true
          cleanup()
          resolve({ success: false, error: 'Authorization window was closed' })
        }
      }
    }, 500)

    // ─── Timeout after 5 minutes ───
    const timeout = setTimeout(() => {
      if (!resolved) {
        resolved = true
        cleanup()
        popupRef.close()
        resolve({ success: false, error: 'Authorization timed out' })
      }
    }, 5 * 60 * 1000)

    function cleanup() {
      window.removeEventListener('message', onMessage)
      clearInterval(pollInterval)
      clearTimeout(timeout)
    }

    window.addEventListener('message', onMessage)
  })
}
