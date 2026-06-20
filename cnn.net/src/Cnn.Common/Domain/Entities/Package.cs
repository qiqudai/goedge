using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("package")]
public class Package
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public int? RegionId { get; set; }

    public int? NodeGroupId { get; set; }

    public int? BackupNodeGroup { get; set; }

    public string? CnameDomain { get; set; }

    public string? CnameHostname2 { get; set; }

    public string? CnameMode { get; set; }

    public int? Traffic { get; set; }

    public string? Bandwidth { get; set; }

    public int? Connection { get; set; }

    public int? Domain { get; set; }

    public int? HttpPort { get; set; }

    public int? StreamPort { get; set; }

    public bool? CustomCcRule { get; set; }

    public bool? Websocket { get; set; }

    public bool? L2Origin { get; set; }

    public DateTime? Expire { get; set; }

    public int? BuyNumLimit { get; set; }

    public string? BackendIpLimit { get; set; }

    public bool? IdVerify { get; set; }

    public int? BeforeExpDaysRenew { get; set; }

    public long? MonthPrice { get; set; }

    public long? QuarterPrice { get; set; }

    public long? YearPrice { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public int? Sort { get; set; }

    public string? Owner { get; set; }

    public bool? Enable { get; set; }
}
