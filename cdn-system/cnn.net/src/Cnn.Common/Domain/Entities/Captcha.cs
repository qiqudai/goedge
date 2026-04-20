using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("captcha")]
public class Captcha
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Email { get; set; }

    public string? Phone { get; set; }

    [SugarColumn(ColumnName = "captcha")]
    public string? CaptchaCode { get; set; }

    public string? Ip { get; set; }

    public DateTime? CreateAt { get; set; }
}
