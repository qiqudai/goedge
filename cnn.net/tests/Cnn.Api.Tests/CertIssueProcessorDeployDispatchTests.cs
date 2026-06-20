using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Cnn.Common.Contracts;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging.Abstractions;
using SqlSugar;
using SystemTask = System.Threading.Tasks.Task;
using TaskEntity = Cnn.Domain.Entities.Task;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class CertIssueProcessorDeployDispatchTests
{
    [Fact]
    public async SystemTask ProcessAsync_DispatchesDueDeployCertTaskToConnectedNode()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-dispatch-" + Guid.NewGuid().ToString("N"),
            State = "waiting",
            Enable = true,
            Data = "{\"cert_id\":88,\"cert\":\"a\",\"key\":\"b\",\"domains\":[\"a.example.com\"]}",
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var connections = new TestConnectionManager("11");

        var sut = BuildSut(scope.Db, connections);
        await sut.ProcessAsync(CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("running", task.State);
        Assert.NotNull(task.StartAt);
        Assert.NotNull(task.TargetsJson);
        Assert.Contains("\"11\"", task.TargetsJson);
        Assert.Contains("\"state\":\"running\"", task.TargetsJson);

        var sent = connections.SentPayloads;
        Assert.NotEmpty(sent);
        using var doc = JsonDocument.Parse(sent[0]);
        Assert.Equal("task_dispatch", doc.RootElement.GetProperty("kind").GetString());
        Assert.Equal(taskId, doc.RootElement.GetProperty("task").GetProperty("task_id").GetInt64());
        Assert.Equal(AgentTaskTypes.DeployCert, doc.RootElement.GetProperty("task").GetProperty("task_type").GetString());
    }

    [Fact]
    public async SystemTask ProcessAsync_DispatchesDeployCertTaskToAllConnectedTargets()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-dispatch-all-" + Guid.NewGuid().ToString("N"),
            State = "waiting",
            Enable = true,
            Data = "{\"cert_id\":108,\"cert\":\"a\",\"key\":\"b\",\"domains\":[\"all.example.com\"]}",
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var connections = new TestConnectionManager("11", "12");
        var sut = BuildSut(scope.Db, connections);
        await sut.ProcessAsync(CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("running", task.State);
        Assert.NotNull(task.TargetsJson);
        Assert.Contains("\"11\"", task.TargetsJson);
        Assert.Contains("\"12\"", task.TargetsJson);
        Assert.Equal(2, connections.SentPayloads.Count);
    }

    [Fact]
    public async SystemTask ProcessAsync_DoesNotDispatchRetryingDeployCertTaskBeforeRetryAt()
    {
        using var scope = new RealMySqlTestScope();
        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-dispatch-delay-" + Guid.NewGuid().ToString("N"),
            State = "retrying",
            Enable = true,
            RetryAt = DateTime.Now.AddMinutes(8),
            Data = "{\"cert_id\":99,\"cert\":\"a\",\"key\":\"b\",\"domains\":[\"b.example.com\"]}",
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var connections = new TestConnectionManager("11");

        var sut = BuildSut(scope.Db, connections);
        await sut.ProcessAsync(CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("retrying", task.State);
        Assert.Empty(connections.SentPayloads);
    }

    [Fact]
    public async SystemTask ProcessAsync_DeployCertStrictPolicy_PartialPermanentFailureTransitionsToFail()
    {
        using var scope = new RealMySqlTestScope();
        var targets = new TaskTargets
        {
            Nodes = new Dictionary<string, TaskTarget>(StringComparer.Ordinal)
            {
                ["11"] = new() { State = TaskTargetState.Success, Tries = 1 },
                ["12"] = new() { State = TaskTargetState.FailedFinal, Tries = 3, Ret = "deploy failed" }
            }
        }.Marshal();

        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-strict-partial-fail-" + Guid.NewGuid().ToString("N"),
            State = "retrying",
            Enable = true,
            TargetsJson = targets,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = BuildSut(scope.Db, new TestConnectionManager("11", "12"));
        await sut.ProcessAsync(CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("fail", task.State);
        Assert.Contains("strict policy", task.Ret ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async SystemTask ProcessAsync_DeployCertTolerantPolicy_PartialPermanentFailureTransitionsToSuccess()
    {
        using var scope = new RealMySqlTestScope();
        await scope.Db.Insertable(new Cnn.Domain.Entities.Config
        {
            Name = DeployCertCompletionPolicy.ConfigKey,
            Value = DeployCertCompletionPolicy.AllowPartialFailures,
            Type = "system",
            ScopeName = "global",
            ScopeId = 0,
            Enable = true,
            CreateAt = DateTime.Now
        }).ExecuteCommandAsync();

        var targets = new TaskTargets
        {
            Nodes = new Dictionary<string, TaskTarget>(StringComparer.Ordinal)
            {
                ["11"] = new() { State = TaskTargetState.Success, Tries = 1 },
                ["12"] = new() { State = TaskTargetState.FailedFinal, Tries = 3, Ret = "deploy failed" }
            }
        }.Marshal();

        var taskId = await scope.Db.Insertable(new TaskEntity
        {
            Type = AgentTaskTypes.DeployCert,
            Name = "deploy-tolerant-partial-success-" + Guid.NewGuid().ToString("N"),
            State = "retrying",
            Enable = true,
            TargetsJson = targets,
            CreateAt = DateTime.Now
        }).ExecuteReturnBigIdentityAsync();

        var sut = BuildSut(scope.Db, new TestConnectionManager("11", "12"));
        await sut.ProcessAsync(CancellationToken.None);

        var task = await scope.Db.Queryable<TaskEntity>().Where(t => t.Id == taskId).SingleAsync();
        Assert.Equal("success", task.State);
        Assert.Contains("partial success", task.Ret ?? string.Empty, StringComparison.OrdinalIgnoreCase);
    }

    private static CertIssueProcessor BuildSut(ISqlSugarClient db, IAgentConnectionManager connections)
    {
        var config = new ConfigurationBuilder()
            .AddInMemoryCollection(new Dictionary<string, string?>
            {
                ["App:SecretKey"] = "0123456789abcdef0123456789abcdef"
            })
            .Build();

        return new CertIssueProcessor(
            db,
            connections,
            new NodeStatusService(),
            new NodeRateLimitService(),
            new CryptoService(config),
            new NoopConfigVersionService(),
            config,
            new SystemConfigService(db),
            NullLogger<CertIssueProcessor>.Instance);
    }

    private sealed class NoopConfigVersionService : IConfigVersionService
    {
        public Task<long> BumpAsync(string resource, IReadOnlyList<long> ids, CancellationToken cancellationToken)
        {
            return Task.FromResult(1L);
        }
    }

    private sealed class TestConnectionManager : IAgentConnectionManager
    {
        private readonly HashSet<string> _nodeIds = new(StringComparer.OrdinalIgnoreCase);
        private readonly WebSocket _socket = new OpenSocket();
        private readonly List<string> _sentPayloads = new();

        public TestConnectionManager(params string[] nodeIds)
        {
            foreach (var nodeId in nodeIds)
            {
                if (!string.IsNullOrWhiteSpace(nodeId))
                {
                    _nodeIds.Add(nodeId.Trim());
                }
            }
        }

        public IReadOnlyList<string> SentPayloads => _sentPayloads;

        public void Register(string connectionId, WebSocket socket)
        {
        }

        public void BindNode(string connectionId, string nodeId)
        {
        }

        public bool TryGetNodeId(string connectionId, out string nodeId)
        {
            nodeId = string.Empty;
            return false;
        }

        public void Remove(string connectionId)
        {
        }

        public bool TryGetSocket(string nodeId, out WebSocket socket)
        {
            socket = _socket;
            return _nodeIds.Contains(nodeId);
        }

        public IReadOnlyCollection<string> GetConnectedNodeIds()
        {
            return _nodeIds.ToList();
        }

        public Task SendAsync(WebSocket socket, object payload, CancellationToken cancellationToken)
        {
            _sentPayloads.Add(JsonSerializer.Serialize(payload));
            return Task.CompletedTask;
        }

        private sealed class OpenSocket : WebSocket
        {
            public override WebSocketCloseStatus? CloseStatus => null;
            public override string? CloseStatusDescription => null;
            public override WebSocketState State => WebSocketState.Open;
            public override string? SubProtocol => null;
            public override void Abort() { }
            public override Task CloseAsync(WebSocketCloseStatus closeStatus, string? statusDescription, CancellationToken cancellationToken) => Task.CompletedTask;
            public override Task CloseOutputAsync(WebSocketCloseStatus closeStatus, string? statusDescription, CancellationToken cancellationToken) => Task.CompletedTask;
            public override void Dispose() { }
            public override Task<WebSocketReceiveResult> ReceiveAsync(ArraySegment<byte> buffer, CancellationToken cancellationToken)
                => Task.FromResult(new WebSocketReceiveResult(0, WebSocketMessageType.Text, true));
            public override ValueTask<ValueWebSocketReceiveResult> ReceiveAsync(Memory<byte> buffer, CancellationToken cancellationToken)
                => ValueTask.FromResult(new ValueWebSocketReceiveResult(0, WebSocketMessageType.Text, true));
            public override Task SendAsync(ArraySegment<byte> buffer, WebSocketMessageType messageType, bool endOfMessage, CancellationToken cancellationToken)
                => Task.CompletedTask;
            public override ValueTask SendAsync(ReadOnlyMemory<byte> buffer, WebSocketMessageType messageType, bool endOfMessage, CancellationToken cancellationToken)
                => ValueTask.CompletedTask;
            public override ValueTask SendAsync(ReadOnlyMemory<byte> buffer, WebSocketMessageType messageType, WebSocketMessageFlags messageFlags, CancellationToken cancellationToken)
                => ValueTask.CompletedTask;
        }
    }
}
