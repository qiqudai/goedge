using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using Cnn.Common.Contracts;

namespace Cnn.Api.Cache;

public static class CacheConfigCompiler
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull,
        PropertyNameCaseInsensitive = true
    };

    public static CacheSiteConfigDto Compile(Site site, CacheConfigDto? input)
    {
        var config = input ?? new CacheConfigDto();
        var profiles = EnsureProfiles(config.Profiles);
        var rules = ExpandRules(config.Rules, site);
        var hosts = CollectHosts(site, rules);

        return new CacheSiteConfigDto
        {
            SiteId = site.Id,
            Version = site.Version ?? (int)DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
            Hosts = hosts,
            Profiles = profiles,
            Rules = rules
        };
    }

    public static string Serialize(CacheSiteConfigDto compiled)
    {
        return JsonSerializer.Serialize(compiled, JsonOptions);
    }

    public static CacheConfigDto? DeserializeConfig(string? json)
    {
        if (string.IsNullOrWhiteSpace(json))
        {
            return null;
        }

        return JsonSerializer.Deserialize<CacheConfigDto>(json, JsonOptions);
    }

    public static string ComputeMd5(string input)
    {
        var bytes = Encoding.UTF8.GetBytes(input);
        var hash = MD5.HashData(bytes);
        return Convert.ToHexString(hash).ToLowerInvariant();
    }

    private static Dictionary<string, CacheProfileDto> EnsureProfiles(Dictionary<string, CacheProfileDto>? profiles)
    {
        var result = new Dictionary<string, CacheProfileDto>(StringComparer.OrdinalIgnoreCase);

        if (profiles != null)
        {
            foreach (var pair in profiles)
            {
                if (string.IsNullOrWhiteSpace(pair.Key) || pair.Value == null)
                {
                    continue;
                }

                result[pair.Key] = pair.Value;
            }
        }

        if (!result.ContainsKey(CacheProfiles.Home))
        {
            result[CacheProfiles.Home] = new CacheProfileDto { Ttl = 60, IgnoreQuery = false, ForceCache = false };
        }

        if (!result.ContainsKey(CacheProfiles.Site))
        {
            result[CacheProfiles.Site] = new CacheProfileDto { Ttl = 300, IgnoreQuery = false, ForceCache = false };
        }

        if (!result.ContainsKey(CacheProfiles.Static))
        {
            result[CacheProfiles.Static] = new CacheProfileDto
            {
                Ttl = 2592000,
                IgnoreQuery = true,
                ForceCache = true,
                QueryIgnoreList = new List<string> { "utm_*" }
            };
        }

        if (!result.ContainsKey(CacheProfiles.Video))
        {
            result[CacheProfiles.Video] = new CacheProfileDto { Ttl = 604800, IgnoreQuery = false, ForceCache = true };
        }

        return result;
    }

    private static List<CacheRuleDto> ExpandRules(List<CacheRuleDto>? rules, Site site)
    {
        var result = new List<CacheRuleDto>();
        if (rules == null || rules.Count == 0)
        {
            return result;
        }

        var siteHosts = SplitHosts(site.Domain).ToList();
        if (!string.IsNullOrWhiteSpace(site.CnameDomain))
        {
            siteHosts.AddRange(SplitHosts(site.CnameDomain));
        }

        foreach (var rule in rules)
        {
            if (rule == null)
            {
                continue;
            }

            var profile = string.IsNullOrWhiteSpace(rule.Profile) ? CacheProfiles.Site : rule.Profile;
            if (string.IsNullOrWhiteSpace(rule.Host))
            {
                if (siteHosts.Count == 0)
                {
                    result.Add(new CacheRuleDto
                    {
                        Host = rule.Host,
                        PathPrefix = rule.PathPrefix,
                        PathRegex = rule.PathRegex,
                        Profile = profile
                    });
                }
                else
                {
                    foreach (var host in siteHosts)
                    {
                        result.Add(new CacheRuleDto
                        {
                            Host = host,
                            PathPrefix = rule.PathPrefix,
                            PathRegex = rule.PathRegex,
                            Profile = profile
                        });
                    }
                }
            }
            else
            {
                result.Add(new CacheRuleDto
                {
                    Host = rule.Host.Trim(),
                    PathPrefix = rule.PathPrefix,
                    PathRegex = rule.PathRegex,
                    Profile = profile
                });
            }
        }

        return result;
    }

    private static List<string> CollectHosts(Site site, List<CacheRuleDto> rules)
    {
        var hosts = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var host in SplitHosts(site.Domain))
        {
            hosts.Add(host);
        }

        if (!string.IsNullOrWhiteSpace(site.CnameDomain))
        {
            foreach (var host in SplitHosts(site.CnameDomain))
            {
                hosts.Add(host);
            }
        }

        foreach (var rule in rules)
        {
            if (!string.IsNullOrWhiteSpace(rule.Host))
            {
                hosts.Add(rule.Host.Trim());
            }
        }

        return hosts.ToList();
    }

    private static IEnumerable<string> SplitHosts(string? hosts)
    {
        if (string.IsNullOrWhiteSpace(hosts))
        {
            yield break;
        }

        var trimmed = hosts.Trim();
        if (trimmed.StartsWith('[') && trimmed.EndsWith(']'))
        {
            var parsed = TryParseJsonHosts(trimmed);
            if (parsed != null)
            {
                foreach (var value in parsed)
                {
                    yield return value;
                }

                yield break;
            }
        }

        var parts = hosts.Split(new[] { ',', ';', ' ', '\n', '\r', '\t' }, StringSplitOptions.RemoveEmptyEntries);
        foreach (var part in parts)
        {
            var host = part.Trim();
            if (!string.IsNullOrWhiteSpace(host))
            {
                yield return host;
            }
        }
    }

    private static List<string>? TryParseJsonHosts(string raw)
    {
        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Array)
            {
                return null;
            }

            var result = new List<string>();
            foreach (var item in doc.RootElement.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.String)
                {
                    continue;
                }

                var value = item.GetString()?.Trim();
                if (!string.IsNullOrWhiteSpace(value))
                {
                    result.Add(value);
                }
            }

            return result;
        }
        catch
        {
            return null;
        }
    }
}

public static class CacheProfiles
{
    public const string Home = "Home";
    public const string Site = "Site";
    public const string Static = "Static";
    public const string Video = "Video";
}
