using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("site_conf_cache")]
public class SiteConfCache
{
    public int? SiteId { get; set; }

    public string? TemplMd5 { get; set; }

    public int? Version { get; set; }

    public string? Data { get; set; }
}