namespace Cnn.Api.Services.Authz;

public static class AccessResourceResolver
{
    public static string? ResolveAgentNodeResourceId(HttpContext context)
    {
        if (context == null)
        {
            return null;
        }

        if (context.Items.TryGetValue("node_id", out var itemValue) && itemValue != null)
        {
            var fromItem = itemValue.ToString()?.Trim();
            if (!string.IsNullOrWhiteSpace(fromItem))
            {
                return fromItem;
            }
        }

        if (context.Request.Query.TryGetValue("node_id", out var queryValues))
        {
            var fromQuery = queryValues.ToString().Trim();
            if (!string.IsNullOrWhiteSpace(fromQuery))
            {
                return fromQuery;
            }
        }

        return null;
    }

    public static string? ResolveGeneralResourceId(HttpContext context, string path)
    {
        if (context == null)
        {
            return null;
        }

        if (context.Request.Query.TryGetValue("node_id", out var nodeValues))
        {
            var value = nodeValues.ToString().Trim();
            if (!string.IsNullOrWhiteSpace(value))
            {
                return value;
            }
        }

        if (context.Request.Query.TryGetValue("user_id", out var userValues))
        {
            var value = userValues.ToString().Trim();
            if (!string.IsNullOrWhiteSpace(value))
            {
                return value;
            }
        }

        if (context.Request.Query.TryGetValue("tenant_id", out var tenantValues))
        {
            var value = tenantValues.ToString().Trim();
            if (!string.IsNullOrWhiteSpace(value))
            {
                return value;
            }
        }

        if (string.IsNullOrWhiteSpace(path))
        {
            return null;
        }

        var segments = path.Split('/', StringSplitOptions.RemoveEmptyEntries);
        for (var i = 0; i < segments.Length - 1; i++)
        {
            if (!segments[i].EndsWith("s", StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            var candidate = segments[i + 1].Trim();
            if (candidate.Length == 0)
            {
                continue;
            }

            if (candidate.All(static ch => char.IsLetterOrDigit(ch) || ch == '-' || ch == '_'))
            {
                return candidate;
            }
        }

        return null;
    }
}
