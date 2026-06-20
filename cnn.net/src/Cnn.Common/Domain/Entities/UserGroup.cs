using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("user_group")]
public class UserGroup
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }
}

