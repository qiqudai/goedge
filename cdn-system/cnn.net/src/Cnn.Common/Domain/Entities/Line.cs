using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("line")]
public class Line
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? NodeGroupId { get; set; }

    public int? NodeId { get; set; }

    public int? NodeIpId { get; set; }

    public string? LineId { get; set; }

    public string? LineName { get; set; }

    public string? Weight { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public string? RecordId { get; set; }

    public long? TaskId { get; set; }

    public bool? Enable { get; set; }

    public bool? IsBackup { get; set; }

    public bool? EnableBackup { get; set; }

    public bool? IsBackupDefaultLine { get; set; }

    public bool? EnableBackupDefaultLine { get; set; }

    public DateTime? SwitchAt { get; set; }

    public string? DisableBy { get; set; }
}