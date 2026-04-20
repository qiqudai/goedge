using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("tlock")]
public class Tlock
{
    [SugarColumn(IsPrimaryKey = true)]
    public string? Name { get; set; }
}