using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("merge_site_group")]
public class MergeSiteGroup
{
    public int? SiteId { get; set; }

    public int? GroupId { get; set; }
}