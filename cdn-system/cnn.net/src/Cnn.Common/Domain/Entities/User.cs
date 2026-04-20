using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("user")]
public class User
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public string? Email { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Phone { get; set; }

    public string? Qq { get; set; }

    public string? CertId { get; set; }

    public string? CertName { get; set; }

    public string? CertNo { get; set; }

    public bool? CertVerified { get; set; }

    public string? WhiteIp { get; set; }

    public string? LoginCaptcha { get; set; }

    public long? Balance { get; set; }

    public long? Freeze { get; set; }

    public DateTime? CreateAt { get; set; }

    public string? Password { get; set; }

    public bool? Enable { get; set; }

    public int? Type { get; set; }
}