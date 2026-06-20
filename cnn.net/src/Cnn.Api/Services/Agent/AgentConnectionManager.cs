using System.Collections.Concurrent;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Agent;

public interface IAgentConnectionManager
{
    void Register(string connectionId, WebSocket socket);
    void BindNode(string connectionId, string nodeId);
    bool TryGetNodeId(string connectionId, out string nodeId);
    void Remove(string connectionId);
    bool TryGetSocket(string nodeId, out WebSocket socket);
    IReadOnlyCollection<string> GetConnectedNodeIds();
    Task SendAsync(WebSocket socket, object payload, CancellationToken cancellationToken);
}

public sealed class AgentConnectionManager : IAgentConnectionManager
{
    private readonly ConcurrentDictionary<string, WebSocket> _connections = new(StringComparer.OrdinalIgnoreCase);
    private readonly ConcurrentDictionary<string, string> _nodeByConnection = new(StringComparer.OrdinalIgnoreCase);
    private readonly ConcurrentDictionary<string, WebSocket> _connectionByNode = new(StringComparer.OrdinalIgnoreCase);

    public void Register(string connectionId, WebSocket socket)
    {
        if (string.IsNullOrWhiteSpace(connectionId) || socket == null)
        {
            return;
        }

        _connections[connectionId] = socket;
    }

    public void BindNode(string connectionId, string nodeId)
    {
        if (string.IsNullOrWhiteSpace(connectionId) || string.IsNullOrWhiteSpace(nodeId))
        {
            return;
        }

        if (_connections.TryGetValue(connectionId, out var socket))
        {
            _nodeByConnection[connectionId] = nodeId;
            _connectionByNode[nodeId] = socket;
        }
    }

    public bool TryGetNodeId(string connectionId, out string nodeId)
    {
        nodeId = string.Empty;
        if (string.IsNullOrWhiteSpace(connectionId))
        {
            return false;
        }

        if (!_nodeByConnection.TryGetValue(connectionId, out var resolved) || string.IsNullOrWhiteSpace(resolved))
        {
            return false;
        }

        nodeId = resolved;
        return true;
    }

    public void Remove(string connectionId)
    {
        if (string.IsNullOrWhiteSpace(connectionId))
        {
            return;
        }

        _connections.TryRemove(connectionId, out _);
        if (_nodeByConnection.TryRemove(connectionId, out var nodeId))
        {
            _connectionByNode.TryRemove(nodeId, out _);
        }
    }

    public bool TryGetSocket(string nodeId, out WebSocket socket)
    {
        socket = null!;
        if (string.IsNullOrWhiteSpace(nodeId))
        {
            return false;
        }

        return _connectionByNode.TryGetValue(nodeId, out socket!);
    }

    public IReadOnlyCollection<string> GetConnectedNodeIds()
    {
        return _connectionByNode.Keys.ToList();
    }

    public Task SendAsync(WebSocket socket, object payload, CancellationToken cancellationToken)
    {
        if (socket == null || socket.State != WebSocketState.Open)
        {
            return Task.CompletedTask;
        }

        var json = JsonSerializer.Serialize(payload);
        var bytes = Encoding.UTF8.GetBytes(json);
        return socket.SendAsync(bytes, WebSocketMessageType.Text, true, cancellationToken);
    }
}
