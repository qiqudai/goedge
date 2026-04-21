using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;

namespace Cnn.Api.Services.Agent;

public interface IWsDispatchService
{
    Task<ServiceResult<WsDispatchResponse>> DispatchAsync(WsDispatchRequest request, CancellationToken cancellationToken);
}

public sealed class WsDispatchService : IWsDispatchService
{
    private readonly IAgentConnectionManager _connections;
    private readonly IAgentAckWaiter _waiter;
    private readonly ILogger<WsDispatchService>? _logger;

    public WsDispatchService(
        IAgentConnectionManager connections,
        IAgentAckWaiter waiter,
        ILogger<WsDispatchService>? logger = null)
    {
        _connections = connections;
        _waiter = waiter;
        _logger = logger;
    }

    public async Task<ServiceResult<WsDispatchResponse>> DispatchAsync(WsDispatchRequest request, CancellationToken cancellationToken)
    {
        var nodeId = request.NodeId.GetValueOrDefault();
        var taskType = request.TaskType?.Trim();

        if (nodeId <= 0 || string.IsNullOrWhiteSpace(taskType))
        {
            return ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.InvalidParam);
        }

        if (!AgentTaskTypes.IsSupported(taskType))
        {
            return ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.InvalidParam, "unsupported_task_type");
        }

        taskType = AgentTaskTypes.Normalize(taskType);
        _logger?.LogInformation(
            "ws dispatch request node_id={NodeId} task_type={TaskType}",
            nodeId,
            taskType);

        var response = new WsDispatchResponse
        {
            NodeId = nodeId,
            Connected = false,
            TaskId = 0
        };

        if (!_connections.TryGetSocket(nodeId.ToString(), out var socket))
        {
            response.Error = "node not connected";
            _logger?.LogWarning(
                "ws dispatch node not connected node_id={NodeId} task_type={TaskType}",
                nodeId,
                taskType);
            return ServiceResult<WsDispatchResponse>.FailWithData(ErrorCodes.WsNotConnected, response, "ws_not_connected");
        }

        response.Connected = true;
        var msgId = $"test-{DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()}-{Guid.NewGuid():N}";
        var payload = new
        {
            kind = "task_dispatch",
            msg_id = msgId,
            task = new
            {
                task_id = 0,
                task_type = taskType,
                task_name = "ws-dispatch-test",
                payload = request.Payload ?? string.Empty
            }
        };

        try
        {
            await _connections.SendAsync(socket, payload, cancellationToken);
            _logger?.LogInformation(
                "ws dispatch sent node_id={NodeId} task_type={TaskType} msg_id={MsgId}",
                nodeId,
                taskType,
                msgId);
        }
        catch
        {
            _logger?.LogWarning(
                "ws dispatch send failed node_id={NodeId} task_type={TaskType}",
                nodeId,
                taskType);
            return ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.ServiceUnavailable);
        }

        var waitSeconds = request.WaitSeconds.GetValueOrDefault();
        if (waitSeconds <= 0)
        {
            waitSeconds = 5;
        }

        var ack = await _waiter.WaitAsync(msgId, TimeSpan.FromSeconds(waitSeconds), cancellationToken);
        if (ack == null)
        {
            response.State = "timeout";
            _logger?.LogWarning(
                "ws dispatch ack timeout node_id={NodeId} task_type={TaskType} msg_id={MsgId} wait_seconds={WaitSeconds}",
                nodeId,
                taskType,
                msgId,
                waitSeconds);
            return ServiceResult<WsDispatchResponse>.Ok(response);
        }

        response.TaskId = ack.TaskId;
        response.State = ack.Status;
        response.Error = string.IsNullOrWhiteSpace(ack.Error) ? null : ack.Error;
        _logger?.LogInformation(
            "ws dispatch ack node_id={NodeId} task_type={TaskType} msg_id={MsgId} ack_task_id={AckTaskId} ack_status={AckStatus} ack_error={AckError}",
            nodeId,
            taskType,
            msgId,
            ack.TaskId,
            ack.Status ?? string.Empty,
            response.Error ?? string.Empty);
        return ServiceResult<WsDispatchResponse>.Ok(response);
    }
}
