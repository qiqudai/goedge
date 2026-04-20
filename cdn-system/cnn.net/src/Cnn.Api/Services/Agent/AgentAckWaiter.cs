using System.Collections.Concurrent;

namespace Cnn.Api.Services.Agent;

public interface IAgentAckWaiter
{
    Task<TaskAckMessage?> WaitAsync(string msgId, TimeSpan timeout, CancellationToken cancellationToken);
    void Notify(TaskAckMessage message);
}

public sealed class AgentAckWaiter : IAgentAckWaiter
{
    private readonly ConcurrentDictionary<string, TaskCompletionSource<TaskAckMessage>> _waiters =
        new(StringComparer.OrdinalIgnoreCase);

    public async Task<TaskAckMessage?> WaitAsync(string msgId, TimeSpan timeout, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(msgId))
        {
            return null;
        }

        var tcs = new TaskCompletionSource<TaskAckMessage>(TaskCreationOptions.RunContinuationsAsynchronously);
        if (!_waiters.TryAdd(msgId, tcs))
        {
            return null;
        }

        using var timeoutCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        timeoutCts.CancelAfter(timeout);

        try
        {
            var completed = await Task.WhenAny(tcs.Task, Task.Delay(Timeout.Infinite, timeoutCts.Token));
            if (completed == tcs.Task)
            {
                return await tcs.Task;
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            _waiters.TryRemove(msgId, out _);
        }

        return null;
    }

    public void Notify(TaskAckMessage message)
    {
        if (message == null)
        {
            return;
        }

        var msgId = message.MsgId;
        if (string.IsNullOrWhiteSpace(msgId))
        {
            return;
        }

        if (_waiters.TryRemove(msgId, out var waiter))
        {
            waiter.TrySetResult(message);
        }
    }
}
