using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("site_group")]
public class SiteGroup
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }
}