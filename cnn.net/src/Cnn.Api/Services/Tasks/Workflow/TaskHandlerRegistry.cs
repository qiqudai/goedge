namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class TaskHandlerRegistry : ITaskHandlerRegistry
{
    private readonly IReadOnlyList<ITaskHandler> _handlers;

    public TaskHandlerRegistry(IEnumerable<ITaskHandler> handlers)
    {
        _handlers = handlers.ToList();
    }

    public ITaskHandler Resolve(string taskType)
    {
        if (string.IsNullOrWhiteSpace(taskType))
        {
            throw new ArgumentException("taskType is required.", nameof(taskType));
        }

        var handler = _handlers.FirstOrDefault(x => x.CanHandle(taskType));
        if (handler != null) return handler;

        throw new KeyNotFoundException($"No task handler registered for task type '{taskType}'.");
    }

    public bool TryResolve(string taskType, out ITaskHandler? handler)
    {
        handler = null;
        if (string.IsNullOrWhiteSpace(taskType))
        {
            return false;
        }

        handler = _handlers.FirstOrDefault(x => x.CanHandle(taskType));
        return handler != null;
    }
}
