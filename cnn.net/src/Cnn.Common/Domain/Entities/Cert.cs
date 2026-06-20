using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("cert")]
public class Cert
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Type { get; set; }

    public string? Domain { get; set; }

    public int? Dnsapi { get; set; }

    [SugarColumn(ColumnName = "cert")]
    public string? CertPem { get; set; }

    public string? Key { get; set; }

    public DateTime? StartTime { get; set; }

    public DateTime? ExpireTime { get; set; }

    public bool? AutoRenew { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }

    public long? IssueTaskId { get; set; }

    public string? State { get; set; }

    public string? Ret { get; set; }

    public int? Version { get; set; }
}
