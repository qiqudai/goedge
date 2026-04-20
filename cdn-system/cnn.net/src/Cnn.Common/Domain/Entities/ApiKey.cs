using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("api_key")]
public class ApiKey
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    [SugarColumn(ColumnName = "api_key")]
    public string? ApiKeyValue { get; set; }

    public string? ApiSecret { get; set; }

    public string? ApiIp { get; set; }
}
