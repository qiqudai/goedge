namespace Cnn.Common.Contracts;

public static class AgentMessageKinds
{
    public const string Ack = "ack";
    public const string Heartbeat = "heartbeat";
    public const string HeartbeatAck = "heartbeat_ack";
    public const string NodeSync = "node_sync";
    public const string L2NodesRequest = "l2_nodes_request";
    public const string L2NodesResponse = "l2_nodes_response";
    public const string L2Heartbeat = "l2_heartbeat";
    
    public const string LogsAccess = "agent_logs_access";
    public const string LogsStream = "agent_logs_stream";
    public const string LogsMetrics = "agent_logs_metrics";
    public const string LogsEvents = "agent_logs_events";
    
    public const string CertIssued = "cert_issued";
    public const string TaskAck = "task_ack";
    public const string TaskUpdate = "task_update";
    
    public const string Ping = "ping";
    public const string Pong = "pong";
    
    public const string Hello = "hello";
    public const string AgentHello = "agent_hello";
    
    public const string EdgeConfig = "edge_config";
    public const string CacheConfig = "cache_config";
}
