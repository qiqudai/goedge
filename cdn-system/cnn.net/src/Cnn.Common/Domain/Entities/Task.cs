using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("task")]
public class Task
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public int? Pid { get; set; }

    public int? Pry { get; set; }

    public string? Name { get; set; }

    public string? Type { get; set; }

    public string? Res { get; set; }

    public string? Data { get; set; }

    public string? TargetsJson { get; set; }

    public string? Depend { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? StartAt { get; set; }

    public DateTime? EndAt { get; set; }

    public string? Ret { get; set; }

    public bool? Enable { get; set; }

    public string? State { get; set; }

    public int? ErrTimes { get; set; }

    public DateTime? RetryAt { get; set; }

    public string? Progress { get; set; }
}
