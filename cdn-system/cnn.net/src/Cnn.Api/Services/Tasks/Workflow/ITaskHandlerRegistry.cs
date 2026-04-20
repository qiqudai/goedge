namespace Cnn.Api.Services.Tasks.Workflow;

public interface ITaskHandlerRegistry
{
    ITaskHandler Resolve(string taskType);

    bool TryResolve(string taskType, out ITaskHandler? handler);
}
