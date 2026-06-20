namespace Cnn.Agent.Plugin;

public interface IRulePlugin
{
    string Name { get; }
    string Version { get; }
    ValueTask InitializeAsync(IReadOnlyDictionary<string, string> settings, CancellationToken ct);
    ValueTask<PluginDecision> EvaluateAsync(HttpContext context, CancellationToken ct);
    ValueTask DisposeAsync();
}
