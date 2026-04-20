using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("domains")]
public class Domain
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public long? UserId { get; set; }

    public string? Name { get; set; }

    public string? Cname { get; set; }

    public int? Status { get; set; }

    [SugarColumn(ColumnName = "origins")]
    public string? OriginsRaw { get; set; }

    [SugarColumn(ColumnName = "created_at")]
    public DateTime? CreatedAt { get; set; }

    [SugarColumn(ColumnName = "updated_at")]
    public DateTime? UpdatedAt { get; set; }

    [SugarColumn(IsIgnore = true)]
    public List<DomainOrigin>? Origins { get; set; }
}
