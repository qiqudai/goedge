using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("config")]
public class Config
{
    public string? Name { get; set; }

    public string? Value { get; set; }

    public string? Type { get; set; }

    public int? ScopeId { get; set; }

    public string? ScopeName { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }
}