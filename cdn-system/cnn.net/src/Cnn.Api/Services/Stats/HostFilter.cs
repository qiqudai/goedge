namespace Cnn.Api.Services.Stats;

public sealed class HostFilter
{
    public List<string> Exact { get; } = new();
    public List<string> Wildcards { get; } = new();

    public bool Empty => Exact.Count == 0 && Wildcards.Count == 0;

    public string BuildHttpCondition()
    {
        var conditions = new List<string>();
        if (Exact.Count > 0)
        {
            var quoted = Exact.Select(ClickHouseHttpHelper.QuoteString);
            conditions.Add("host IN (" + string.Join(",", quoted) + ")");
        }

        foreach (var suffix in Wildcards)
        {
            conditions.Add("host LIKE " + ClickHouseHttpHelper.QuoteString("%" + suffix));
        }

        if (conditions.Count == 0)
        {
            return string.Empty;
        }

        return "(" + string.Join(" OR ", conditions) + ")";
    }
}
