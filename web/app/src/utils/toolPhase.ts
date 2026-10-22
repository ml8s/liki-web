/**
 * Maps a raw agent tool name to its i18n label key.
 *
 * The server forwards tool names verbatim (facts only); every user-visible
 * string lives in the frontend locales. Rules are evaluated in order and the
 * first match wins; the list mirrors the server-side phase table removed in P0
 * with one correction: `bazi_liunian` is a period-analysis tool and maps to
 * `period` (the old server table matched it to `chart` by rule order).
 */
type Rule = readonly [match: (name: string) => boolean, key: string]

const RULES: ReadonlyArray<Rule> = [
  [name => name.startsWith('load_skill'), 'chat.tool.expert'],
  [name => name.startsWith('transfer_to_agent'), 'chat.tool.expert'],
  [name => name === 'bazi', 'chat.tool.bazi'],
  [name => name === 'ziwei', 'chat.tool.ziwei'],
  [name => name.startsWith('city_coords'), 'chat.tool.coords'],
  [name => name.startsWith('tianwen_resolve'), 'chat.tool.coords'],
  [name => name.startsWith('bazi_liunian'), 'chat.tool.period'],
  [name => name.startsWith('bazi_'), 'chat.tool.chart'],
  [name => name.startsWith('compute_factors'), 'chat.tool.chart'],
  [name => name.startsWith('natal_query'), 'chat.tool.chart'],
  [name => name.startsWith('period_query'), 'chat.tool.period'],
  [name => name.startsWith('qimen_'), 'chat.tool.qimen'],
  [name => name.startsWith('liuyao_'), 'chat.tool.liuyao'],
  [name => name.startsWith('naming_'), 'chat.tool.naming'],
  [name => name.startsWith('fengshui_'), 'chat.tool.fengshui'],
]

export function toolPhaseKey(tool: string): string {
  const name = tool.trim()
  for (const [match, key] of RULES) {
    if (match(name)) return key
  }
  return 'chat.tool.generic'
}
