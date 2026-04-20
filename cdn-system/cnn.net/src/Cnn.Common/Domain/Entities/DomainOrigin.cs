using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("domain_origins")]
public class DomainOrigin
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public long? DomainId { get; set; }

    public string? Addr { get; set; }

    public int? Port { get; set; }

    public int? Weight { get; set; }

    public string? Protocol { get; set; }

    [SugarColumn(ColumnName = "created_at")]
    public DateTime? CreatedAt { get; set; }

    [SugarColumn(ColumnName = "updated_at")]
    public DateTime? UpdatedAt { get; set; }
}
