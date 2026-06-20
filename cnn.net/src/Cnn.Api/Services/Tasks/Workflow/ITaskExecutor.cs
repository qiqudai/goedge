namespace Cnn.Api.Services.Tasks.Workflow;

public interface ITaskExecutor
{
    Task ExecuteAsync(long taskId, CancellationToken cancellationToken);
}
