using System.Text.Json;
using System.Text.Json.Serialization;

namespace Cnn.Api.Services.Admin;

internal sealed class IssueCertTaskMeta
{
    [JsonPropertyName("target_node_id")]
    public long TargetNodeId { get; set; }

    [JsonPropertyName("local")]
    public bool Local { get; set; }

    public static string Build(long targetNodeId, bool local)
    {
        if (targetNodeId <= 0 && !local)
        {
            return string.Empty;
        }

        var payload = new IssueCertTaskMeta
        {
            TargetNodeId = targetNodeId,
            Local = local
        };

        return JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web));
    }

    public static IssueCertTaskMeta? Parse(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<IssueCertTaskMeta>(raw, new JsonSerializerOptions(JsonSerializerDefaults.Web));
        }
        catch
        {
            return null;
        }
    }
}
