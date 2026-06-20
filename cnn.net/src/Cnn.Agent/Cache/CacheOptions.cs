namespace Cnn.Agent.Cache;

public sealed class CacheOptions
{
    public string Root { get; set; } = "www";

    public Dictionary<string, CacheProfileOptions>? Profiles { get; set; }

    public List<CacheRuleOptions>? Rules { get; set; }

    public List<string>? StaticExtensions { get; set; }

    public List<string>? VideoExtensions { get; set; }

    public List<string>? HomePaths { get; set; }
}

public sealed class CacheProfileOptions
{
    public int Ttl { get; set; }

    public bool IgnoreQuery { get; set; }

    public bool ForceCache { get; set; }

    public List<string>? QueryIgnoreList { get; set; }
}

public sealed class CacheRuleOptions
{
    public string? Host { get; set; }

    public string? PathPrefix { get; set; }

    public string? PathRegex { get; set; }

    public string? Profile { get; set; }
}
