export const originConditionItems = [
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
  { label: 'HTTP版本', value: 'http_version' }
];

export const originConditionOperators = [
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
  { label: '在IP段', value: 'ip_range' },
  { label: '不在IP段', value: 'not_ip_range' }
];

export const cacheTypeLabelMap = {
  index: '首页',
  all: '全站',
  dir: '目录',
  suffix: '后缀',
  path: '路径'
};
