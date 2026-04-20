using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("package_group")]
public class PackageGroup
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public int? Sort { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }
}