using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record RegionListResult(IReadOnlyList<RegionListItem> List, long Total);

public sealed record RegionListItem(
    long Id,
    string? Name,
    string? Remark,
    int? L2CheckPort,
    int? SortOrder,
    DateTime? CreatedAt,
    DateTime? UpdatedAt
);

public sealed class RegionUpsertRequest
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("l2_check_port")]
    public int? L2CheckPort { get; set; }

    [JsonPropertyName("sort_order")]
    public int? SortOrder { get; set; }
}
