using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("op_log")]
public class OpLog
{
    [SugarColumn(ColumnName = "id", IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    [SugarColumn(ColumnName = "uid")]
    public int? Uid { get; set; }

    [SugarColumn(ColumnName = "type")]
    public string? Type { get; set; }

    [SugarColumn(ColumnName = "action")]
    public string? Action { get; set; }

    [SugarColumn(ColumnName = "content")]
    public string? Content { get; set; }

    [SugarColumn(ColumnName = "diff")]
    public string? Diff { get; set; }

    [SugarColumn(ColumnName = "ip")]
    public string? Ip { get; set; }

    [SugarColumn(ColumnName = "create_at")]
    public DateTime? CreateAt { get; set; }

    [SugarColumn(ColumnName = "process")]
    public string? Process { get; set; }
}
