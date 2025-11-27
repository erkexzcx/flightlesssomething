import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime.js'

// Enable relative time plugin
dayjs.extend(relativeTime)

/**
 * Formats a date string into a human-readable relative time format.
 * 
 * @param {string} dateString - ISO 8601 date string
 * @param {string} fallbackText - Text to show when date is missing (default: 'Unknown')
 * @returns {string} Human-readable relative time (e.g., "2 days ago", "a month ago")
 */
export function formatRelativeDate(dateString, fallbackText = 'Unknown') {
  if (!dateString) return fallbackText
  
  const date = dayjs(dateString)
  
  // Check for invalid date
  if (!date.isValid()) return fallbackText
  
  // Use dayjs's fromNow() for relative time formatting
  return date.fromNow()
}
