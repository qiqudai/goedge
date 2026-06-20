namespace Cnn.Agent.Plugin;

public sealed record PluginDecision(
    bool Handled,
    bool Allowed,
    int StatusCode,
    string? Reason);
