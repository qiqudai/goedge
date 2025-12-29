export const DNS_API_FIELD_LABELS = {
  dns_pod: {
    token: 'Token'
  },
  ali_dns: {
    access_key_id: 'AccessKey ID',
    access_key_secret: 'AccessKey Secret'
  },
  cloud_flare: {
    api_key: 'API Key',
    email: 'Email'
  }
}

export const CACHE_RULE_TYPES = [
  { label: '首页', value: 'home' },
  { label: '全站', value: 'all' },
  { label: '目录', value: 'dir' },
  { label: '后缀', value: 'suffix' },
  { label: '单个路径', value: 'path' }
]

export const ORIGIN_CONDITION_ITEMS = [
  { label: '请求URI', value: 'uri' },
  { label: '请求URI(不带参数)', value: 'uri_no_args' },
  { label: '节点国家代码', value: 'node_country' },
  { label: '节点运营商', value: 'node_isp' },
  { label: '节点省份', value: 'node_province' },
  { label: '节点城市', value: 'node_city' },
  { label: '客户端国家代码', value: 'client_country' },
  { label: '客户端运营商', value: 'client_isp' },
  { label: '客户端省份', value: 'client_province' },
  { label: '客户端城市', value: 'client_city' },
  { label: '用户IP', value: 'client_ip' },
  { label: '域名', value: 'domain' },
  { label: '请求头', value: 'header' },
  { label: '请求方法', value: 'method' },
  { label: 'HTTP版本', value: 'http_version' },
  { label: '独立UA数量', value: 'ua_count' },
  { label: '404状态码数量', value: 'status_404' }
]

export const ORIGIN_CONDITION_OPERATORS = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
  { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' },
  { label: '前缀匹配', value: 'prefix' },
  { label: '后缀匹配', value: 'suffix' },
  { label: '正则匹配', value: 'regex' },
  { label: '正则不匹配', value: 'not_regex' },
  { label: '存在', value: 'exists' },
  { label: '不存在', value: 'not_exists' },
  { label: '在IP段', value: 'in_ip' },
  { label: '不在IP段', value: 'not_in_ip' }
]
