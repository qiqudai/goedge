using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("node_monitor_log")]
public class NodeMonitorLog
{
    public DateTime? CreateAt { get; set; }

    public string? Type { get; set; }

    public string? EventId { get; set; }

    public string? Ip { get; set; }

    public string? Success { get; set; }

    public int? NodeId { get; set; }
}