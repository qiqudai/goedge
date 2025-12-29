export const DNS_PROVIDERS = {
  cloudflare: { name: 'Cloudflare', domain: 'cloudflare.com' },
  dnspod: { name: 'DNSPod', domain: 'dnspod.cn' },
  dnspod_intl: { name: 'DNSPod Intl', domain: 'dnspod.com' },
  godaddy: { name: 'GoDaddy', domain: 'godaddy.com' },
  aliyun: { name: 'Aliyun', domain: 'aliyun.com' },
  cloudns: { name: 'ClouDNS', domain: 'cloudns.net' },
  namecom: { name: 'Name.com', domain: 'name.com' },
  namecheap: { name: 'Namecheap', domain: 'namecheap.com' },
  jdcloud: { name: 'JD Cloud', domain: 'jdcloud.com' },
  dnsla: { name: 'DNS.LA', domain: 'dns.la' },
  namesilo: { name: 'Namesilo', domain: 'namesilo.com' },
  '51dns': { name: '51DNS', domain: '51dns.com' },
  huawei: { name: 'Huawei Cloud', domain: 'huaweicloud.com' }
}

export const DNS_PROVIDER_LABEL_MAP = {
  aliyun: 'Aliyun (aliyun.com, alidns.aliyun.com)',
  huawei: 'Huawei Cloud (huaweicloud.com)',
  dnsla: 'DNS.LA (dns.la)',
  dnspod: 'DNSPod (dnspod.cn)',
  dnspod_intl: 'DNSPod Intl (dnspod.com)',
  '51dns': '51DNS (51dns.com)',
  cloudflare: 'Cloudflare (cloudflare.com)',
  godaddy: 'GoDaddy (godaddy.com)',
  cloudns: 'ClouDNS (cloudns.net)',
  namecom: 'Name.com (name.com)',
  namecheap: 'Namecheap (namecheap.com)',
  jdcloud: 'JD Cloud (jdcloud.com)',
  namesilo: 'Namesilo (namesilo.com)'
}

export const DNS_API_FIELD_LABELS = {
  aliyun: { access_key_id: 'AccessKey ID', access_key_secret: 'AccessKey Secret' },
  huawei: { id: 'Access Key ID', secret: 'Secret Access Key' },
  dnsla: { id: 'API ID', secret: 'API Password' },
  dnspod: { id: 'ID', token: 'Token' },
  dnspod_intl: { secret_id: 'SecretId', secret_key: 'SecretKey' },
  '51dns': { id: 'App ID', secret: 'App Secret' },
  cloudflare: { email: 'Email', key: 'API Key' },
  godaddy: { key: 'API Key', secret: 'API Secret' },
  cloudns: { auth_id: 'Auth ID', auth_password: 'Auth Password' },
  namecom: { username: 'Username', api_token: 'API Token' },
  namecheap: { user: 'User', api_key: 'API Key', ip: 'Client IP' },
  jdcloud: { access_key: 'Access Key', secret_key: 'Secret Key' },
  namesilo: { api_key: 'API Key' }
}

