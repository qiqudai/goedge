namespace Cnn.Api.Services.Tasks.Workflow;

public interface ITaskCommandFactory
{
    Task<TaskRequestResult> CreateAsync(CreateTaskCommand command, CancellationToken cancellationToken);
}
