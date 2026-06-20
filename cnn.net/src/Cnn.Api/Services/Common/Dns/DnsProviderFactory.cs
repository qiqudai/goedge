using Cnn.Api.Services.Common.Dns.Providers;

namespace Cnn.Api.Services.Common.Dns;

public static class DnsProviderFactory
{
    public static IDnsRecordProvider? TryCreate(string? type, string? credentials)
    {
        if (string.IsNullOrWhiteSpace(type) || string.IsNullOrWhiteSpace(credentials))
        {
            return null;
        }

        var key = type.Trim().ToLowerInvariant();
        return key switch
        {
            "aliyun" or "aliyun.com" => AliyunDnsProvider.TryCreate(credentials),
            "cloudflare" or "cloudflare.com" => CloudflareDnsProvider.TryCreate(credentials),
            "dnspod" or "dnspod.com" or "dnspod.cn" => DnsPodProvider.TryCreate(credentials, false),
            "dnspod_intl" => DnsPodProvider.TryCreate(credentials, true),
            "dnsla" or "dns.la" => DnsLaProvider.TryCreate(credentials),
            "huawei" or "huaweicloud.com" => HuaweiDnsProvider.TryCreate(credentials),
            "godaddy" => GoDaddyDnsProvider.TryCreate(credentials),
            "namecom" or "name.com" => NameComDnsProvider.TryCreate(credentials),
            "namecheap" => NamecheapDnsProvider.TryCreate(credentials),
            "cloudns" or "cloudns.net" => ClouDnsProvider.TryCreate(credentials),
            "namesilo" or "namesilo.com" => NamesiloDnsProvider.TryCreate(credentials),
            "jdcloud" or "jdcloud.com" => JDCloudDnsProvider.TryCreate(credentials),
            "51dns" or "51dns.com" => Dns51Provider.TryCreate(credentials),
            _ => null
        };
    }
}
