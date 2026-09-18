import { notify } from './notify'
import { tg } from '../i18n/translations'

/**
 * Copy text to clipboard and show a toast notification.
 */
export async function copyWithToast(text: string, successMsg?: string) {
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      // Fallback: create temporary textarea for copy
      const el = document.createElement('textarea')
      el.value = text
      el.style.position = 'fixed'
      el.style.left = '-9999px'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }
    notify.success(successMsg ?? tg('lib.copied'))
    return true
  } catch (err) {
    console.error('Clipboard copy failed:', err)
    notify.error(tg('lib.copyFailed'))
    return false
  }
}

export default { copyWithToast }
