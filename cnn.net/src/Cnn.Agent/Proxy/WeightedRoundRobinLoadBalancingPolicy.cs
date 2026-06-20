using System.Collections.Concurrent;
using System.Globalization;
using Yarp.ReverseProxy.LoadBalancing;
using Yarp.ReverseProxy.Model;

namespace Cnn.Agent.Proxy;

public sealed class WeightedRoundRobinLoadBalancingPolicy : ILoadBalancingPolicy
{
    public const string PolicyName = "WeightedRoundRobin";

    private readonly ConcurrentDictionary<string, WeightedClusterState> _states = new(StringComparer.OrdinalIgnoreCase);

    public string Name => PolicyName;

    public DestinationState? PickDestination(
        HttpContext context,
        ClusterState cluster,
        IReadOnlyList<DestinationState> availableDestinations)
    {
        if (availableDestinations == null || availableDestinations.Count == 0)
        {
            return null;
        }

        if (availableDestinations.Count == 1)
        {
            return availableDestinations[0];
        }

        var clusterId = cluster.ClusterId ?? "cluster";
        var signature = BuildSignature(availableDestinations);
        var state = _states.AddOrUpdate(
            clusterId,
            _ => BuildState(signature, availableDestinations),
            (_, existing) => existing.Signature == signature ? existing : BuildState(signature, availableDestinations));

        if (state.TotalWeight <= 0 || state.Destinations.Length == 0)
        {
            var idx = Math.Abs(Environment.TickCount) % availableDestinations.Count;
            return availableDestinations[idx];
        }

        var ticket = Interlocked.Increment(ref state.Counter);
        var value = (int)(((ticket - 1) % state.TotalWeight) + 1);
        var selectedIndex = Array.BinarySearch(state.PrefixSums, value);
        if (selectedIndex < 0)
        {
            selectedIndex = ~selectedIndex;
        }

        if (selectedIndex < 0 || selectedIndex >= state.Destinations.Length)
        {
            return state.Destinations[0];
        }

        return state.Destinations[selectedIndex];
    }

    private static WeightedClusterState BuildState(string signature, IReadOnlyList<DestinationState> destinations)
    {
        var weighted = new List<(DestinationState Destination, int Weight)>(destinations.Count);
        var total = 0;
        foreach (var destination in destinations)
        {
            var weight = ReadWeight(destination);
            weighted.Add((destination, weight));
            total += weight;
        }

        if (weighted.Count == 0 || total <= 0)
        {
            return new WeightedClusterState(signature, [], [], 0);
        }

        var prefix = new int[weighted.Count];
        var resolved = new DestinationState[weighted.Count];
        var running = 0;
        for (var i = 0; i < weighted.Count; i++)
        {
            running += weighted[i].Weight;
            prefix[i] = running;
            resolved[i] = weighted[i].Destination;
        }

        return new WeightedClusterState(signature, resolved, prefix, running);
    }

    private static string BuildSignature(IReadOnlyList<DestinationState> destinations)
    {
        var parts = new string[destinations.Count];
        for (var i = 0; i < destinations.Count; i++)
        {
            var dest = destinations[i];
            var weight = ReadWeight(dest);
            parts[i] = $"{dest.DestinationId}:{weight}";
        }

        return string.Join("|", parts);
    }

    private static int ReadWeight(DestinationState destination)
    {
        var metadata = destination.Model?.Config?.Metadata;
        if (metadata != null
            && metadata.TryGetValue("weight", out var raw)
            && int.TryParse(raw, NumberStyles.Integer, CultureInfo.InvariantCulture, out var parsed)
            && parsed > 0)
        {
            return parsed;
        }

        return 1;
    }

    private sealed class WeightedClusterState
    {
        public WeightedClusterState(
            string signature,
            DestinationState[] destinations,
            int[] prefixSums,
            int totalWeight)
        {
            Signature = signature;
            Destinations = destinations;
            PrefixSums = prefixSums;
            TotalWeight = totalWeight;
        }

        public string Signature { get; }
        public DestinationState[] Destinations { get; }
        public int[] PrefixSums { get; }
        public int TotalWeight { get; }
        public long Counter;
    }
}
