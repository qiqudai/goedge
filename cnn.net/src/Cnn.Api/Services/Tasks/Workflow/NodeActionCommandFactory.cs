using System.Text.Json;
using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public static class NodeActionCommandFactory
{
    public static RequestActionCommand CreateStatusChange(long nodeId, bool enable)
    {
        if (nodeId <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(nodeId));
        }

        return new RequestActionCommand
        {
            TaskType = enable ? AsyncTaskTypes.NodeEnable : AsyncTaskTypes.NodeDisable,
            ResourceType = ResourceTypes.Node,
            ResourceId = nodeId,
            DedupeKey = $"node-status:{nodeId}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.Node,
                resource_id = nodeId,
                enable
            })
        };
    }
}
