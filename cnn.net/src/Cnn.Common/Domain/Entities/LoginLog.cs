using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("login_log")]
public class LoginLog
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Ip { get; set; }

    public DateTime? CreateAt { get; set; }

    public bool? Success { get; set; }

    public string? PostContent { get; set; }
}
