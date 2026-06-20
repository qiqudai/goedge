using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("package_up")]
public class PackageUp
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Type { get; set; }

    public int? Amount { get; set; }

    public string? BindPackage { get; set; }

    public long? Price { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Enable { get; set; }
}