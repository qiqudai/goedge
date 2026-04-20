using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("cname_domains")]
public class CnameDomains
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string Domain { get; set; } = string.Empty;

    public int DnsProviderId { get; set; }

    public string? Note { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }
}
