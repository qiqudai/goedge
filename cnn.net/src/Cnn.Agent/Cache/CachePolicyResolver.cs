using System.Collections.Concurrent;
using System.Text.RegularExpressions;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Cache;

public sealed class CachePolicyResolver
{
    private readonly CacheOptions _options;
    private readonly IReadOnlyDictionary<string, CacheProfileOptions> _profiles;
    private readonly IReadOnlyList<CacheRuleMatcher> _rules;
    private readonly HashSet<string> _staticExtensions;
    private readonly HashSet<string> _videoExtensions;
    private readonly HashSet<string> _homePaths;

    public CachePolicyResolver(IOptions<CacheOptions> options)
    {
        _options = options.Value ?? new CacheOptions();
        _profiles = BuildProfiles(_options);
        _rules = BuildRules(_options.Rules);
        _staticExtensions = new HashSet<string>(GetExtensions(_options.StaticExtensions, DefaultStaticExtensions), StringComparer.OrdinalIgnoreCase);
        _videoExtensions = new HashSet<string>(GetExtensions(_options.VideoExtensions, DefaultVideoExtensions), StringComparer.OrdinalIgnoreCase);
        _homePaths = new HashSet<string>(GetHomePaths(_options.HomePaths), StringComparer.OrdinalIgnoreCase);
    }

    public CacheDecision Resolve(HttpContext context)
    {
        var request = context.Request;
        var host = request.Host.Host ?? string.Empty;
        var path = request.Path.Value ?? "/";

        var profileName = ResolveProfileName(host, path);
        if (!_profiles.TryGetValue(profileName, out var profile))
        {
            profileName = CacheProfiles.Site;
            profile = _profiles[CacheProfiles.Site];
        }

        var ttl = TimeSpan.FromSeconds(profile.Ttl <= 0 ? 0 : profile.Ttl);
        return new CacheDecision(profileName, ttl, profile.IgnoreQuery, profile.ForceCache, profile.QueryIgnoreList ?? new List<string>());
    }

    private string ResolveProfileName(string host, string path)
    {
        foreach (var rule in _rules)
        {
            if (rule.IsMatch(host, path))
            {
                return rule.Profile;
            }
        }

        var extension = Path.GetExtension(path);
        if (!string.IsNullOrWhiteSpace(extension))
        {
            if (_videoExtensions.Contains(extension))
            {
                return CacheProfiles.Video;
            }

            if (_staticExtensions.Contains(extension))
            {
                return CacheProfiles.Static;
            }
        }

        if (_homePaths.Contains(path))
        {
            return CacheProfiles.Home;
        }

        return CacheProfiles.Site;
    }

    private static IReadOnlyDictionary<string, CacheProfileOptions> BuildProfiles(CacheOptions options)
    {
        var profiles = new Dictionary<string, CacheProfileOptions>(StringComparer.OrdinalIgnoreCase)
        {
            [CacheProfiles.Home] = new CacheProfileOptions { Ttl = 60, IgnoreQuery = false, ForceCache = false },
            [CacheProfiles.Site] = new CacheProfileOptions { Ttl = 300, IgnoreQuery = false, ForceCache = false },
            [CacheProfiles.Static] = new CacheProfileOptions { Ttl = 2592000, IgnoreQuery = true, ForceCache = true, QueryIgnoreList = new List<string> { "utm_*" } },
            [CacheProfiles.Video] = new CacheProfileOptions { Ttl = 604800, IgnoreQuery = false, ForceCache = true }
        };

        if (options.Profiles == null)
        {
            return profiles;
        }

        foreach (var pair in options.Profiles)
        {
            if (string.IsNullOrWhiteSpace(pair.Key) || pair.Value == null)
            {
                continue;
            }
            profiles[pair.Key] = pair.Value;
        }

        return profiles;
    }

    private static IReadOnlyList<CacheRuleMatcher> BuildRules(IReadOnlyList<CacheRuleOptions>? rules)
    {
        if (rules == null || rules.Count == 0)
        {
            return Array.Empty<CacheRuleMatcher>();
        }

        var list = new List<CacheRuleMatcher>();
        foreach (var rule in rules)
        {
            if (string.IsNullOrWhiteSpace(rule.Profile))
            {
                continue;
            }

            list.Add(new CacheRuleMatcher(rule));
        }

        return list;
    }

    private static IEnumerable<string> GetExtensions(IReadOnlyList<string>? configured, IReadOnlyList<string> defaults)
    {
        if (configured != null && configured.Count > 0)
        {
            return configured;
        }

        return defaults;
    }

    private static IEnumerable<string> GetHomePaths(IReadOnlyList<string>? configured)
    {
        if (configured != null && configured.Count > 0)
        {
            return configured;
        }

        return DefaultHomePaths;
    }

    private sealed class CacheRuleMatcher
    {
        private readonly string? _host;
        private readonly string? _pathPrefix;
        private readonly Regex? _pathRegex;

        public CacheRuleMatcher(CacheRuleOptions rule)
        {
            _host = string.IsNullOrWhiteSpace(rule.Host) ? null : rule.Host.Trim();
            _pathPrefix = string.IsNullOrWhiteSpace(rule.PathPrefix) ? null : rule.PathPrefix.Trim();
            _pathRegex = string.IsNullOrWhiteSpace(rule.PathRegex) ? null : new Regex(rule.PathRegex, RegexOptions.IgnoreCase | RegexOptions.Compiled);
            Profile = rule.Profile ?? CacheProfiles.Site;
        }

        public string Profile { get; }

        public bool IsMatch(string host, string path)
        {
            if (_host != null && !string.Equals(_host, host, StringComparison.OrdinalIgnoreCase))
            {
                return false;
            }

            if (_pathPrefix != null && !path.StartsWith(_pathPrefix, StringComparison.OrdinalIgnoreCase))
            {
                return false;
            }

            if (_pathRegex != null && !_pathRegex.IsMatch(path))
            {
                return false;
            }

            return true;
        }
    }

    private static readonly IReadOnlyList<string> DefaultStaticExtensions = new[]
    {
        ".css", ".js", ".map", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
        ".woff", ".woff2", ".ttf", ".otf", ".eot",
        ".txt", ".xml", ".json", ".pdf",
        ".zip", ".rar", ".7z",
        ".mp3", ".wav",
        ".wasm"
    };

    private static readonly IReadOnlyList<string> DefaultVideoExtensions = new[]
    {
        ".mp4", ".m4v", ".mkv", ".mov", ".avi", ".flv", ".webm", ".ts", ".m3u8"
    };

    private static readonly IReadOnlyList<string> DefaultHomePaths = new[]
    {
        "/", "/index.html", "/index.htm"
    };
}

public static class CacheProfiles
{
    public const string Home = "Home";
    public const string Site = "Site";
    public const string Static = "Static";
    public const string Video = "Video";
}

public sealed record CacheDecision(string Profile, TimeSpan Ttl, bool IgnoreQuery, bool ForceCache, IReadOnlyList<string> QueryIgnoreList)
{
    public bool Enabled => Ttl > TimeSpan.Zero;
}
