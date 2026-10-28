/**
 * Build-time flags are deliberately string comparisons: an unset or malformed
 * value must keep the legacy App as the safe default.
 */
export const appFlowRedesignEnabled =
  import.meta.env.VITE_FEATURE_APP_FLOW_REDESIGN === 'true'
