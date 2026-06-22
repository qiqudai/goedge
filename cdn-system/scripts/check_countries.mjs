import fs from 'fs'

const content = fs.readFileSync('web/admin/src/components/CountrySelector.vue', 'utf8')
const codes = [...content.matchAll(/value: '([a-z]{2})'/g)].map((m) => m[1])
const byRegion = {}
let current = ''
for (const line of content.split('\n')) {
  if (line.includes("label: '亚洲'")) current = '亚洲'
  if (line.includes("label: '非洲'")) current = '非洲'
  if (line.includes("label: '欧洲'")) current = '欧洲'
  if (line.includes("label: '北美洲'")) current = '北美洲'
  if (line.includes("label: '南美洲'")) current = '南美洲'
  if (line.includes("label: '大洋洲'")) current = '大洋洲'
  const m = line.match(/value: '([a-z]{2})'/)
  if (m && current) {
    byRegion[current] = byRegion[current] || []
    byRegion[current].push(m[1])
  }
}

const africa = [
  'dz', 'ao', 'bj', 'bw', 'bf', 'bi', 'cv', 'cm', 'cf', 'td', 'km', 'cg', 'cd', 'ci', 'dj', 'eg', 'gq', 'er', 'sz', 'et', 'ga', 'gm', 'gh', 'gn', 'gw', 'ke', 'ls', 'lr', 'ly', 'mg', 'mw', 'ml', 'mr', 'mu', 'ma', 'mz', 'na', 'ne', 'ng', 'rw', 'st', 'sn', 'sc', 'sl', 'so', 'za', 'ss', 'sd', 'tz', 'tg', 'tn', 'ug', 'zm', 'zw'
]
const asia = [
  'af', 'am', 'az', 'bh', 'bd', 'bt', 'bn', 'kh', 'cn', 'ge', 'in', 'id', 'ir', 'iq', 'jp', 'jo', 'kz', 'kp', 'kr', 'kw', 'kg', 'la', 'lb', 'my', 'mv', 'mn', 'mm', 'np', 'om', 'pk', 'ps', 'ph', 'qa', 'ru', 'sa', 'sg', 'lk', 'sy', 'tw', 'tj', 'th', 'tm', 'tr', 'ae', 'uz', 'vn', 'ye', 'hk', 'mo', 'tl', 'cy', 'il'
]
const europe = [
  'al', 'ad', 'at', 'by', 'be', 'ba', 'bg', 'hr', 'cy', 'cz', 'dk', 'ee', 'fi', 'fr', 'de', 'gr', 'hu', 'is', 'ie', 'it', 'lv', 'li', 'lt', 'lu', 'mt', 'md', 'mc', 'me', 'nl', 'mk', 'no', 'pl', 'pt', 'ro', 'ru', 'sm', 'rs', 'sk', 'si', 'es', 'se', 'ch', 'tr', 'ua', 'gb', 'va'
]
const northAmerica = ['us', 'ca', 'mx', 'bz', 'cr', 'cu', 'do', 'gt', 'ht', 'hn', 'jm', 'ni', 'pa', 'bs', 'bb', 'tt', 'gl', 'ag', 'dm', 'gd', 'kn', 'lc', 'vc', 'aw']
const southAmerica = ['ar', 'bo', 'br', 'cl', 'co', 'ec', 'gy', 'py', 'pe', 'sr', 'uy', 've', 'gf']
const oceania = ['au', 'fj', 'ki', 'mh', 'nr', 'nz', 'pw', 'pg', 'ws', 'sb', 'to', 'tv', 'vu', 'fm']

const lua = fs.readFileSync('agent/assets/lua/geo_country.lua', 'utf8')
const luaEntries = [...lua.matchAll(/\["([^"]+)"\]\s*=\s*"([A-Z]{2})"/g)].map((m) => ({ label: m[1], code: m[2].toLowerCase() }))
const luaCodes = new Set(luaEntries.map((e) => e.code))

const regionConfig = fs.readFileSync('web/admin/src/components/RegionConfig.vue', 'utf8')
const rcCountries = [...regionConfig.matchAll(/countries: \[([^\]]+)\]/g)].flatMap((m) =>
  m[1].split(',').map((s) => s.trim().replace(/^'|'$/g, ''))
)

function missing(list, name) {
  const miss = list.filter((c) => !codes.includes(c))
  console.log(`${name} missing (${miss.length}):`, miss.join(', '))
  return miss
}

console.log('CountrySelector total:', codes.length)
console.log('Africa in selector:', byRegion['非洲']?.length)
missing(africa, 'Africa')
missing(asia, 'Asia')
missing(europe, 'Europe')
missing(northAmerica, 'North America')
missing(southAmerica, 'South America')
missing(oceania, 'Oceania')

const selectorNotLua = codes.filter((c) => !luaCodes.has(c))
console.log('Selector codes missing in geo_country.lua:', selectorNotLua.join(', '))

const luaNotSelector = [...luaCodes].filter((c) => !codes.includes(c)).sort()
console.log('geo_country codes missing in selector:', luaNotSelector.join(', '))

// RegionConfig vs CountrySelector labels
const selectorLabels = [...content.matchAll(/\{ label: '([^']+)', value: '([a-z]{2})' \}/g)].map((m) => m[1])
const rcOnly = rcCountries.filter((c) => !selectorLabels.includes(c))
console.log('RegionConfig countries not in CountrySelector labels:', rcOnly.join(', '))
