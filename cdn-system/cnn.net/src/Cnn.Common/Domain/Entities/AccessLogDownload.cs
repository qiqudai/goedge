using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("access_log_download")]
public class AccessLogDownload
{
    [SugarColumn(ColumnName = "id", IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    [SugarColumn(ColumnName = "user_id")]
    public long? UserId { get; set; }

    [SugarColumn(ColumnName = "is_admin")]
    public bool? IsAdmin { get; set; }

    [SugarColumn(ColumnName = "scope")]
    public string? Scope { get; set; }

    [SugarColumn(ColumnName = "state")]
    public string? State { get; set; }

    [SugarColumn(ColumnName = "query_json")]
    public string? QueryJson { get; set; }

    [SugarColumn(ColumnName = "file_name")]
    public string? FileName { get; set; }

    [SugarColumn(ColumnName = "rows")]
    public long? Rows { get; set; }

    [SugarColumn(ColumnName = "error")]
    public string? Error { get; set; }

    [SugarColumn(ColumnName = "create_at")]
    public DateTime? CreateAt { get; set; }

    [SugarColumn(ColumnName = "finish_at")]
    public DateTime? FinishAt { get; set; }
}
