using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("res_count")]
public class ResCount
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public DateTime? Time { get; set; }

    public int? UserPackage { get; set; }

    public int? Uid { get; set; }

    public string? Cate { get; set; }

    public string? Type { get; set; }

    public string? Res { get; set; }

    public long? Value { get; set; }
}