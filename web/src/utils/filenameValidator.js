/**
 * Utility functions for validating filenames
 */

/**
 * Checks if a filename contains a date/time pattern like:
 * - cs2_2025-12-21_12-40-08
 * - benchmark_2024_01_15_14_30_45
 * - test-2023-06-30-09-15-30
 * 
 * Pattern matches: YYYY[-_]MM[-_]DD[-_]HH[-_]MM[-_]SS
 * Year is validated to be in range 20XX (2000-2099)
 * 
 * @param {string} filename - The filename to check
 * @returns {boolean} - True if the filename contains a date/time pattern
 */
export function hasDateTimePattern(filename) {
  if (!filename || typeof filename !== 'string') {
    return false;
  }
  
  // Match YYYY[-_]MM[-_]DD[-_]HH[-_]MM[-_]SS pattern
  // Year: 20\d{2} (2000-2099), Month: \d{2}, Day: \d{2}, Hour: \d{2}, Minute: \d{2}, Second: \d{2}
  // Separator: [-_]
  const dateTimePattern = /20\d{2}[-_]\d{2}[-_]\d{2}[-_]\d{2}[-_]\d{2}[-_]\d{2}/;
  
  return dateTimePattern.test(filename);
}

/**
 * Checks if any of the filenames in an array contains a date/time pattern
 * 
 * @param {Array<string>} filenames - Array of filenames to check
 * @returns {boolean} - True if any filename contains a date/time pattern
 */
export function hasAnyDateTimePattern(filenames) {
  if (!Array.isArray(filenames)) {
    return false;
  }
  
  return filenames.some(filename => hasDateTimePattern(filename));
}

/**
 * Generates a user-friendly warning message for filenames with date/time patterns
 * 
 * @returns {string} - The warning message
 */
export function getDateTimeWarningMessage() {
  return 'It looks way nicer if you remove date/time from the filenames. ' +
         'For example, "cs2_2025-12-21_12-40-08 - Async Off" should be replaced with ' +
         '"Async Off" (assuming game name is provided in the title or description).';
}
