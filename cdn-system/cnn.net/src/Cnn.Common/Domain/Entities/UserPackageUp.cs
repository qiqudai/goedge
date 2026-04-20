using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("user_package_up")]
public class UserPackageUp
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public int? PackageUp { get; set; }

    public int? UserPackage { get; set; }

    public int? Amount { get; set; }
}