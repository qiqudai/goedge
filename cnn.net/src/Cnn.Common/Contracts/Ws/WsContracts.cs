namespace Cnn.Common.Contracts.Ws;

public static class WsKinds
{
    public const string Hello = "hello";
    public const string Ack = "ack";
    public const string Ping = "ping";
    public const string Pong = "pong";
    public const string RouteOpen = "route_open";
    public const string RouteClose = "route_close";
    public const string Data = "data";
}

public record WsEnvelope(string Kind, object Payload);

public record WsHello(string NodeId, string Token, string Version);

public record WsAck(string ConnectionId, int HeartbeatSeconds);

public record WsPing(long Ts);

public record WsPong(long Ts);

public record WsRouteOpen(string StreamId, string TargetHost, int TargetPort);

public record WsRouteClose(string StreamId, string Reason);

public record WsData(string StreamId, byte[] Payload);
