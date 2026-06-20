using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("line_delete_queue")]
public class LineDeleteQueue
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? NodeId { get; set; }

    public int? NodeGroupId { get; set; }

    public string? LineId { get; set; }

    public string? LineName { get; set; }

    public DateTime? DeleteAt { get; set; }

    [SugarColumn(ColumnName = "create_at")]
    public DateTime? CreatedAt { get; set; }
}
