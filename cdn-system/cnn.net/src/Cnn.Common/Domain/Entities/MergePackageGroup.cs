using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("merge_package_group")]
public class MergePackageGroup
{
    public int? PackageId { get; set; }

    public int? PackageGroupId { get; set; }
}